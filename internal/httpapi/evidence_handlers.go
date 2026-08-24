package httpapi

import (
	"net/http"
	"strconv"

	"task215-fiberlineage/internal/model"
)

func (s *Server) registerEvidenceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/evidence", s.handleAddEvidence)
	mux.HandleFunc("GET /api/evidence", s.handleListEvidence)
	mux.HandleFunc("PUT /api/evidence/{id}", s.handleUpdateEvidence)
	mux.HandleFunc("POST /api/evidence/{id}/validate", s.handleValidateEvidence)
	mux.HandleFunc("POST /api/evidence/{id}/exclude", s.handleExcludeEvidence)
}

func (s *Server) handleAddEvidence(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SampleA int64   `json:"sample_a"`
		SampleB int64   `json:"sample_b"`
		Kind    string  `json:"kind"`
		Weight  float64 `json:"weight"`
		Note    string  `json:"note"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	ev, err := s.svc.AddEvidence(req.SampleA, req.SampleB, model.EvidenceKind(req.Kind), req.Weight, req.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ev)
}

func (s *Server) handleListEvidence(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	status := r.URL.Query().Get("status")
	list, err := s.svc.ListEvidence(kind, status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"evidence": list, "count": len(list)})
}

func (s *Server) handleUpdateEvidence(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	var req struct {
		Weight float64 `json:"weight"`
		Note   string  `json:"note"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	ev, err := s.svc.UpdateEvidence(id, req.Weight, req.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (s *Server) handleValidateEvidence(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if req.Status == string(model.EvidenceValid) {
		req.Status = string(model.EvidencePending)
	}
	ev, err := s.svc.ValidateEvidence(id, model.EvidenceStatus(req.Status))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (s *Server) handleExcludeEvidence(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	ev, err := s.svc.ExcludeEvidence(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev)
}
