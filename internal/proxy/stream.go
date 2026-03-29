package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/YoungsoonLee/aegis/internal/audit"
	"github.com/YoungsoonLee/aegis/internal/guard"
)

// isStreamingRequest checks if the request body contains "stream": true.
func isStreamingRequest(body []byte) bool {
	var req struct {
		Stream bool `json:"stream"`
	}
	if json.Unmarshal(body, &req) != nil {
		return false
	}
	return req.Stream
}

// handleStreaming forwards an SSE streaming request to the upstream target
// and relays chunks to the client in real time with immediate flushing.
// After the stream completes, it runs outbound guard checks on the
// accumulated response text and logs a complete audit event.
func (p *Proxy) handleStreaming(w http.ResponseWriter, r *http.Request, t *target, body []byte, guardResults []*guard.Result, start time.Time) {
	upstreamURL := buildUpstreamURL(t.url, r.URL)
	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		p.logger.Error("failed to create upstream streaming request", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	upstreamReq.Header = r.Header.Clone()
	upstreamReq.Header.Del("Accept-Encoding")
	upstreamReq.ContentLength = int64(len(body))

	resp, err := (&http.Client{}).Do(upstreamReq)
	if err != nil {
		p.logger.Error("upstream streaming request failed", "error", err)
		p.errorHandler(w, r, err)
		return
	}
	defer resp.Body.Close()

	if !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		copyResponse(w, resp)
		p.logStreamAuditEvent(r, t.name, guardResults, resp.StatusCode, "", start)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		p.logger.Error("response writer does not support flushing")
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(resp.StatusCode)

	var accumulated strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if _, err := fmt.Fprintf(w, "%s\n", line); err != nil {
			break
		}
		flusher.Flush()

		if strings.HasPrefix(line, "data: ") {
			data := line[6:]
			if data != "[DONE]" {
				if content := parseSSEDelta(data); content != "" {
					accumulated.WriteString(content)
				}
			}
		}
	}

	fullResponse := accumulated.String()
	if fullResponse != "" {
		p.checkStreamingResponse(r, t.name, fullResponse)
	}

	p.logStreamAuditEvent(r, t.name, guardResults, resp.StatusCode, fullResponse, start)
}

// parseSSEDelta extracts the text content from an SSE data chunk
// in OpenAI-compatible streaming format (choices[0].delta.content).
func parseSSEDelta(data string) string {
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if json.Unmarshal([]byte(data), &chunk) == nil && len(chunk.Choices) > 0 {
		return chunk.Choices[0].Delta.Content
	}
	return ""
}

// checkStreamingResponse runs outbound guards on the accumulated streaming
// response text. Since the text has already been sent to the client, violations
// are logged as warnings for monitoring and audit purposes.
func (p *Proxy) checkStreamingResponse(r *http.Request, targetName, fullResponse string) {
	outContent := &guard.Content{
		Direction: guard.DirectionOutbound,
		Body:      fullResponse,
		Metadata: map[string]string{
			"method":    r.Method,
			"path":      r.URL.Path,
			"target":    targetName,
			"streaming": "true",
		},
	}

	results, err := p.guardEngine.Process(r.Context(), outContent)
	if err != nil {
		p.logger.Error("outbound streaming guard error", "error", err)
		return
	}

	for _, result := range results {
		if result.Action != guard.ActionPass {
			p.logger.Warn("streaming response guard violation",
				"guard", result.GuardName,
				"action", result.Action,
				"details", result.Details,
			)
		}
	}
}

func (p *Proxy) logStreamAuditEvent(r *http.Request, targetName string, guardResults []*guard.Result, statusCode int, responseText string, start time.Time) {
	grs := make([]audit.GuardResult, len(guardResults))
	for i, gr := range guardResults {
		grs[i] = audit.GuardResult{
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
		Guards:    grs,
		Blocked:   false,
		Duration:  time.Since(start),
		Request: audit.RequestInfo{
			ContentLength: r.ContentLength,
		},
		Response: &audit.ResponseInfo{
			StatusCode:    statusCode,
			ContentLength: int64(len(responseText)),
		},
	}

	p.auditLogger.Log(event)
}

func buildUpstreamURL(base *url.URL, reqURL *url.URL) string {
	u := *base
	if base.Path == "" || base.Path == "/" {
		u.Path = reqURL.Path
	} else {
		u.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(reqURL.Path, "/")
	}
	u.RawQuery = reqURL.RawQuery
	return u.String()
}

func copyResponse(w http.ResponseWriter, resp *http.Response) {
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
