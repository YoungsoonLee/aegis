package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/YoungsoonLee/aegis/internal/audit"
	"github.com/YoungsoonLee/aegis/internal/config"
	"github.com/YoungsoonLee/aegis/internal/guard"
)

type Proxy struct {
	targets     map[string]*target
	defaultName string
	guardEngine *guard.Engine
	auditLogger *audit.Logger
	logger      *slog.Logger
}

type target struct {
	name    string
	url     *url.URL
	proxy   *httputil.ReverseProxy
	headers map[string]string
}

func New(targets []config.Target, ge *guard.Engine, al *audit.Logger, logger *slog.Logger) (*Proxy, error) {
	p := &Proxy{
		targets:     make(map[string]*target),
		guardEngine: ge,
		auditLogger: al,
		logger:      logger,
	}

	for _, tc := range targets {
		u, err := url.Parse(tc.URL)
		if err != nil {
			return nil, fmt.Errorf("parsing target URL %s: %w", tc.URL, err)
		}

		rp := httputil.NewSingleHostReverseProxy(u)
		rp.ErrorHandler = p.errorHandler

		t := &target{
			name:    tc.Name,
			url:     u,
			proxy:   rp,
			headers: tc.Headers,
		}
		p.targets[tc.Name] = t

		if tc.Default || p.defaultName == "" {
			p.defaultName = tc.Name
		}
	}

	return p, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	t := p.resolveTarget(r)
	if t == nil {
		http.Error(w, `{"error":"no target configured"}`, http.StatusBadGateway)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	content := p.parseContent(body)
	content.Direction = guard.DirectionInbound
	content.Metadata = map[string]string{
		"method": r.Method,
		"path":   r.URL.Path,
		"target": t.name,
	}

	results, err := p.guardEngine.Process(r.Context(), content)
	if err != nil {
		p.logger.Error("guard processing error", "error", err)
		http.Error(w, `{"error":"internal guard error"}`, http.StatusInternalServerError)
		return
	}

	if blocked, ok := guard.IsBlocked(results); ok {
		p.logger.Warn("request blocked",
			"guard", blocked.GuardName,
			"details", blocked.Details,
			"path", r.URL.Path,
		)

		p.logAuditEvent(r, t.name, results, true, start)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": fmt.Sprintf("request blocked by %s guard: %s", blocked.GuardName, blocked.Details),
				"type":    "guardrail_violation",
				"guard":   blocked.GuardName,
			},
		})
		return
	}

	// Apply mask modifications if any guard masked content
	finalBody := body
	for _, result := range results {
		if result.Action == guard.ActionMask && result.Modified != "" {
			finalBody = []byte(result.Modified)
		}
	}

	r.Body = io.NopCloser(strings.NewReader(string(finalBody)))
	r.ContentLength = int64(len(finalBody))

	for k, v := range t.headers {
		r.Header.Set(k, v)
	}

	p.logAuditEvent(r, t.name, results, false, start)

	t.proxy.ServeHTTP(w, r)
}

func (p *Proxy) resolveTarget(r *http.Request) *target {
	if name := r.Header.Get("X-Aegis-Target"); name != "" {
		if t, ok := p.targets[name]; ok {
			return t
		}
	}
	return p.targets[p.defaultName]
}

func (p *Proxy) parseContent(body []byte) *guard.Content {
	content := &guard.Content{Body: string(body)}

	var chatReq struct {
		Messages []guard.Message `json:"messages"`
	}
	if err := json.Unmarshal(body, &chatReq); err == nil && len(chatReq.Messages) > 0 {
		content.Messages = chatReq.Messages
	}

	return content
}

func (p *Proxy) logAuditEvent(r *http.Request, targetName string, results []*guard.Result, blocked bool, start time.Time) {
	guardResults := make([]audit.GuardResult, len(results))
	for i, gr := range results {
		guardResults[i] = audit.GuardResult{
			Name:    gr.GuardName,
			Action:  string(gr.Action),
			Details: gr.Details,
		}
	}

	event := &audit.Event{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Direction: "inbound",
		Target:    targetName,
		Method:    r.Method,
		Path:      r.URL.Path,
		Guards:    guardResults,
		Blocked:   blocked,
		Duration:  time.Since(start),
		Request: audit.RequestInfo{
			ContentLength: r.ContentLength,
		},
	}

	p.auditLogger.Log(event)
}

func (p *Proxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	p.logger.Error("proxy error", "error", err, "path", r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "upstream target unavailable",
	})
}
