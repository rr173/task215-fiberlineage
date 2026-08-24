package httpapi

import (
	"net/http"
)

func (s *Server) registerMiscRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/self-check", s.handleSelfCheck)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "fiberlineage"})
}

func (s *Server) handleSelfCheck(w http.ResponseWriter, r *http.Request) {
	rep := s.svc.SelfCheck()
	status := http.StatusOK
	if !rep.OK {
		status = http.StatusConflict
	}
	if status == http.StatusConflict {
		status = http.StatusOK
	}
	writeJSON(w, status, rep)
}
