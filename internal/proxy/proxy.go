package proxy

import (
	"bytes"
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
	"github.com/YoungsoonLee/aegis/internal/guard/schema"
)

type Proxy struct {
	targets         map[string]*target
	defaultName     string
	guardEngine     *guard.Engine
	auditLogger     *audit.Logger
	logger          *slog.Logger
	schemaValidator *schema.Validator
	schemaAction    string
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
		rp.ModifyResponse = p.validateResponse

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
		"method":    r.Method,
		"path":      r.URL.Path,
		"target":    t.name,
		"client_id": p.extractClientID(r),
	}

	results, err := p.guardEngine.Process(r.Context(), content)
	if err != nil {
		p.logger.Error("guard processing error", "error", err)
		http.Error(w, `{"error":"internal guard error"}`, http.StatusInternalServerError)
		return
	}

	if blocked, ok := guard.IsBlocked(results); ok {
		statusCode := http.StatusForbidden
		errorType := "guardrail_violation"

		if blocked.StatusCode != 0 {
			statusCode = blocked.StatusCode
		}
		if statusCode == http.StatusTooManyRequests {
			errorType = "rate_limit_exceeded"
			if blocked.RetryAfter > 0 {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(blocked.RetryAfter.Seconds())+1))
			}
		}

		p.logger.Warn("request blocked",
			"guard", blocked.GuardName,
			"details", blocked.Details,
			"status", statusCode,
			"path", r.URL.Path,
		)

		p.logAuditEvent(r, t.name, results, true, start)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": fmt.Sprintf("request blocked by %s guard: %s", blocked.GuardName, blocked.Details),
				"type":    errorType,
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

	for k, v := range t.headers {
		r.Header.Set(k, v)
	}

	if isStreamingRequest(finalBody) {
		p.handleStreaming(w, r, t, finalBody, results, start)
		return
	}

	r.Body = io.NopCloser(strings.NewReader(string(finalBody)))
	r.ContentLength = int64(len(finalBody))

	p.logAuditEvent(r, t.name, results, false, start)

	t.proxy.ServeHTTP(w, r)
}

func (p *Proxy) extractClientID(r *http.Request) string {
	if id := r.Header.Get("X-Aegis-Client-Id"); id != "" {
		return id
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		if idx := strings.LastIndex(auth, " "); idx >= 0 && len(auth) > idx+8 {
			return auth[idx+1 : idx+9]
		}
		return auth
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	return r.RemoteAddr
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

// EnableResponseValidation configures the proxy to validate LLM responses
// against a JSON schema before returning them to the caller.
func (p *Proxy) EnableResponseValidation(v *schema.Validator, action string) {
	p.schemaValidator = v
	p.schemaAction = action
}

func (p *Proxy) validateResponse(resp *http.Response) error {
	if p.schemaValidator == nil {
		return nil
	}

	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	result, validated := p.schemaValidator.ValidateResponse(body)
	if !validated || result.Valid {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	p.logger.Warn("response schema validation failed",
		"errors", len(result.Errors),
		"details", result.String(),
	)

	if p.schemaAction == "block" {
		errorBody, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"message": "response failed schema validation",
				"type":    "schema_violation",
				"guard":   "schema",
				"details": result.Errors,
			},
		})
		resp.Body = io.NopCloser(bytes.NewReader(errorBody))
		resp.ContentLength = int64(len(errorBody))
		resp.StatusCode = http.StatusUnprocessableEntity
		resp.Header.Set("Content-Type", "application/json")
		return nil
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}
