package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/YoungsoonLee/aegis/internal/audit"
	"github.com/YoungsoonLee/aegis/internal/guard"
	"github.com/YoungsoonLee/aegis/internal/policy"
)

type Server struct {
	*http.Server
	guardEngine  *guard.Engine
	policyEngine *policy.Engine
	auditLogger  *audit.Logger
	startTime    time.Time
}

func NewServer(addr string, ge *guard.Engine, pe *policy.Engine, al *audit.Logger) *Server {
	s := &Server{
		guardEngine:  ge,
		policyEngine: pe,
		auditLogger:  al,
		startTime:    time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /readyz", s.handleReady)
	mux.HandleFunc("GET /api/v1/guards", s.handleListGuards)
	mux.HandleFunc("GET /api/v1/policies", s.handleListPolicies)
	mux.HandleFunc("POST /api/v1/policies/reload", s.handleReloadPolicies)

	s.Server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "healthy",
		"uptime": time.Since(s.startTime).String(),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleListGuards(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"guards": s.guardEngine.Guards(),
	})
}

func (s *Server) handleListPolicies(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"policies": s.policyEngine.GetPolicies(),
	})
}

func (s *Server) handleReloadPolicies(w http.ResponseWriter, _ *http.Request) {
	if err := s.policyEngine.Reload(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
