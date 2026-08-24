package httpapi

import (
	"net/http"

	"task215-fiberlineage/internal/model"
)

func (s *Server) registerCompareRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/compare", s.handleCompare)
	mux.HandleFunc("GET /api/similarity", s.handleListSimilarity)
	mux.HandleFunc("POST /api/similarity/recompute", s.handleRecomputeAll)
}

func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SampleA int64 `json:"sample_a"`
		SampleB int64 `json:"sample_b"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	edge, err := s.svc.ComputeSimilarity(req.SampleA, req.SampleB)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, edge)
}

func (s *Server) handleListSimilarity(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Query().Get("method")
	if method == "" {
		method = "weighted"
	}
	edges, err := s.svc.ListSimilarity(method)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"edges": edges, "count": len(edges)})
}

func (s *Server) handleRecomputeAll(w http.ResponseWriter, r *http.Request) {
	n, err := s.svc.RecomputeAllSimilarities()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recomputed_pairs": n})
}
