package audit

import "time"

type Event struct {
	ID        string         `json:"id"`
	Timestamp time.Time      `json:"timestamp"`
	Direction string         `json:"direction"`
	Agent     string         `json:"agent,omitempty"`
	Target    string         `json:"target"`
	Method    string         `json:"method"`
	Path      string         `json:"path"`
	Guards    []GuardResult  `json:"guards"`
	Blocked   bool           `json:"blocked"`
	Duration  time.Duration  `json:"duration_ms"`
	Request   RequestInfo    `json:"request"`
	Response  *ResponseInfo  `json:"response,omitempty"`
}

type GuardResult struct {
	Name    string `json:"name"`
	Action  string `json:"action"`
	Details string `json:"details,omitempty"`
}

type RequestInfo struct {
	ContentLength int64             `json:"content_length"`
	Headers       map[string]string `json:"headers,omitempty"`
}

type ResponseInfo struct {
	StatusCode    int   `json:"status_code"`
	ContentLength int64 `json:"content_length"`
}
