package httpapi

import (
	"net/http"
	"strconv"

	"task215-fiberlineage/internal/model"
)

func (s *Server) registerLineageRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/lineages", s.handleCreateLineage)
	mux.HandleFunc("GET /api/lineages", s.handleListLineages)
	mux.HandleFunc("GET /api/lineages/{id}", s.handleGetLineage)
	mux.HandleFunc("POST /api/lineages/{id}/edges", s.handleAddLineageEdge)
	mux.HandleFunc("POST /api/lineage-edges/{id}/split", s.handleSplitLineageEdge)
	mux.HandleFunc("POST /api/lineages/{id}/merge", s.handleMergeLineages)
	mux.HandleFunc("POST /api/lineages/{id}/confirm", s.handleConfirmLineage)
	mux.HandleFunc("POST /api/lineages/{id}/reject", s.handleRejectLineage)
	mux.HandleFunc("POST /api/lineages/{id}/mutual-exclusive", s.handleMutualExclusive)
	mux.HandleFunc("POST /api/lineages/{id}/counterexamples", s.handleAddCounterexample)
	mux.HandleFunc("GET /api/lineages/{id}/counterexamples", s.handleListCounterexamples)
}

func (s *Server) handleCreateLineage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code      string `json:"code"`
		VersionID int64  `json:"version_id"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	h, err := s.svc.CreateLineage(req.Code, req.VersionID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, h)
}

func (s *Server) handleListLineages(w http.ResponseWriter, r *http.Request) {
	vstr := r.URL.Query().Get("version_id")
	var vid int64
	if vstr != "" {
		var err error
		if vid, err = strconv.ParseInt(vstr, 10, 64); err != nil {
			writeError(w, model.ErrInvalidInput)
			return
		}
	}
	list, err := s.svc.ListLineages(vid)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lineages": list, "count": len(list)})
}

func (s *Server) handleGetLineage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	h, edges, cxs, err := s.svc.GetLineage(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hypothesis": h, "edges": edges, "counterexamples": cxs})
}

func (s *Server) handleAddLineageEdge(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	var req struct {
		SampleA int64  `json:"sample_a"`
		SampleB int64  `json:"sample_b"`
		Relation string `json:"relation"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if !model.ValidLineageRelation(req.Relation) {
		writeError(w, model.ErrInvalidInput)
		return
	}
	edge, err := s.svc.AddLineageEdge(id, req.SampleA, req.SampleB, model.LineageRelation(req.Relation))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, edge)
}

func (s *Server) handleSplitLineageEdge(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if err := s.svc.SplitLineageEdge(id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"edge_id": id, "status": "split"})
}

func (s *Server) handleMergeLineages(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	var req struct {
		SrcID int64 `json:"src_id"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if err := s.svc.MergeLineages(id, req.SrcID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dst_id": id, "src_id": req.SrcID, "status": "merged"})
}

func (s *Server) handleConfirmLineage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if err := s.svc.ConfirmLineage(id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "confirmed"})
}

func (s *Server) handleRejectLineage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if err := s.svc.RejectLineage(id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "rejected"})
}

func (s *Server) handleMutualExclusive(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if err := s.svc.SetMutualExclusive(id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "mutual_exclusive"})
}

func (s *Server) handleAddCounterexample(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	var req struct {
		SampleA int64  `json:"sample_a"`
		SampleB int64  `json:"sample_b"`
		Note    string `json:"note"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	c, err := s.svc.AddCounterexample(id, req.SampleA, req.SampleB, req.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleListCounterexamples(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	list, err := s.svc.ListCounterexamples(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"counterexamples": list, "count": len(list)})
}
