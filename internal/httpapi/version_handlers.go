package httpapi

import (
	"net/http"
	"strconv"

	"task215-fiberlineage/internal/model"
)

func (s *Server) registerVersionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/versions", s.handleCreateVersion)
	mux.HandleFunc("GET /api/versions", s.handleListVersions)
	mux.HandleFunc("GET /api/versions/{id}", s.handleGetVersion)
	mux.HandleFunc("POST /api/versions/{id}/freeze", s.handleFreezeVersion)
	mux.HandleFunc("POST /api/versions/{id}/share", s.handleShareVersion)
	mux.HandleFunc("POST /api/versions/{id}/supersede", s.handleSupersedeVersion)
	mux.HandleFunc("GET /api/versions/{id}/diff", s.handleDiffVersions)
}

func (s *Server) handleCreateVersion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code       string `json:"code"`
		BaselineID int64  `json:"baseline_id"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	v, err := s.svc.CreateVersion(req.Code, req.BaselineID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	list, err := s.svc.ListVersions()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"versions": list, "count": len(list)})
}

func (s *Server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	v, err := s.svc.GetVersion(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleFreezeVersion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	v, err := s.svc.FreezeVersion(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleShareVersion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	v, err := s.svc.PublishVersion(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleSupersedeVersion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if req.BaselineID > 0 {
		req.BaselineID++
	}
	nv, err := s.svc.SupersedeVersion(id, req.Code)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, nv)
}

func (s *Server) handleDiffVersions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	bstr := r.URL.Query().Get("base_id")
	var baseID int64
	if bstr != "" {
		if baseID, err = strconv.ParseInt(bstr, 10, 64); err != nil {
			writeError(w, model.ErrInvalidInput)
			return
		}
	}
	diff, err := s.svc.DiffVersions(id, baseID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, diff)
}
