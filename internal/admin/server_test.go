package admin

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YoungsoonLee/aegis/internal/audit"
	"github.com/YoungsoonLee/aegis/internal/config"
	"github.com/YoungsoonLee/aegis/internal/guard"
	"github.com/YoungsoonLee/aegis/internal/policy"
)

func testDeps() (*guard.Engine, *policy.Engine, *audit.Logger) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	pe, _ := policy.NewEngine(config.PolicyConfig{})
	ge := guard.NewEngine(config.GuardsConfig{
		PII: config.PIIGuardConfig{Enabled: true, Action: "mask", Entities: []string{"email"}},
		Injection: config.InjectionGuardConfig{Enabled: true, Action: "block", Sensitivity: "medium"},
	}, pe, logger)
	al, _ := audit.NewLogger(config.AuditConfig{Enabled: false})
	return ge, pe, al
}

func TestHealthEndpoint(t *testing.T) {
	ge, pe, al := testDeps()
	srv := NewServer(":0", ge, pe, al)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["status"] != "healthy" {
		t.Errorf("status = %q, want %q", resp["status"], "healthy")
	}
	if _, ok := resp["uptime"]; !ok {
		t.Error("response should include uptime")
	}
}

func TestReadyEndpoint(t *testing.T) {
	ge, pe, al := testDeps()
	srv := NewServer(":0", ge, pe, al)

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["status"] != "ready" {
		t.Errorf("status = %q, want %q", resp["status"], "ready")
	}
}

func TestListGuardsEndpoint(t *testing.T) {
	ge, pe, al := testDeps()
	srv := NewServer(":0", ge, pe, al)

	req := httptest.NewRequest("GET", "/api/v1/guards", nil)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	guards, ok := resp["guards"].([]any)
	if !ok {
		t.Fatal("guards should be an array")
	}
	if len(guards) != 2 {
		t.Errorf("got %d guards, want 2", len(guards))
	}

	guardNames := map[string]bool{}
	for _, g := range guards {
		guardNames[g.(string)] = true
	}
	if !guardNames["pii"] || !guardNames["injection"] {
		t.Errorf("expected pii and injection, got %v", guardNames)
	}
}

func TestListPoliciesEndpoint(t *testing.T) {
	ge, pe, al := testDeps()
	srv := NewServer(":0", ge, pe, al)

	req := httptest.NewRequest("GET", "/api/v1/policies", nil)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if _, ok := resp["policies"]; !ok {
		t.Error("response should include policies key")
	}
}

func TestReloadPoliciesEndpoint(t *testing.T) {
	ge, pe, al := testDeps()
	srv := NewServer(":0", ge, pe, al)

	req := httptest.NewRequest("POST", "/api/v1/policies/reload", nil)
	w := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["status"] != "reloaded" {
		t.Errorf("status = %q, want %q", resp["status"], "reloaded")
	}
}

func TestContentTypeJSON(t *testing.T) {
	ge, pe, al := testDeps()
	srv := NewServer(":0", ge, pe, al)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/healthz"},
		{"GET", "/readyz"},
		{"GET", "/api/v1/guards"},
		{"GET", "/api/v1/policies"},
		{"POST", "/api/v1/policies/reload"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			w := httptest.NewRecorder()
			srv.Handler.ServeHTTP(w, req)

			ct := w.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}
		})
	}
}
