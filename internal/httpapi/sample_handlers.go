package httpapi

import (
	"net/http"
	"strconv"

	"task215-fiberlineage/internal/model"
)

type registerSampleReq struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	Source string `json:"source"`
}

type detectReq struct {
	FiberSpectrum []model.FiberBand   `json:"fiber_spectrum"`
	DyePeaks      []model.DyePeak     `json:"dye_peaks"`
	RepairLayers  []model.RepairLayer `json:"repair_layers"`
	UnitKnown     bool                `json:"unit_known"`
}

func (s *Server) registerSampleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/samples", s.handleRegisterSample)
	mux.HandleFunc("GET /api/samples", s.handleListSamples)
	mux.HandleFunc("GET /api/samples/{id}", s.handleGetSample)
	mux.HandleFunc("PUT /api/samples/{id}", s.handleUpdateSample)
	mux.HandleFunc("POST /api/samples/{id}/detect", s.handleSubmitDetection)
	mux.HandleFunc("GET /api/samples/{id}/spectrum", s.handleGetDetection)
	mux.HandleFunc("POST /api/samples/{id}/seal", s.handleSealSample)
}

func (s *Server) handleRegisterSample(w http.ResponseWriter, r *http.Request) {
	var req registerSampleReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	sm, err := s.svc.RegisterSample(req.Code, req.Title, req.Source)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sm)
}

func (s *Server) handleListSamples(w http.ResponseWriter, r *http.Request) {
	samples, err := s.svc.ListSamples()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"samples": samples, "count": len(samples)})
}

func (s *Server) handleGetSample(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	sm, err := s.svc.GetSample(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sm)
}

func (s *Server) handleUpdateSample(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	var req struct {
		Title  string `json:"title"`
		Source string `json:"source"`
		Status string `json:"status"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	sm, err := s.svc.UpdateSample(id, req.Title, req.Source, req.Status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sm)
}

func (s *Server) handleSubmitDetection(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	var req detectReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	det := &model.SampleDetection{
		FiberSpectrum: req.FiberSpectrum,
		DyePeaks:      req.DyePeaks,
		RepairLayers:  req.RepairLayers,
		UnitKnown:     req.UnitKnown,
	}
	if err := s.svc.SubmitDetection(id, det); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sample_id": id, "status": "comparable"})
}

func (s *Server) handleGetDetection(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	det, err := s.svc.GetDetection(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, det)
}

func (s *Server) handleSealSample(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	sm, err := s.svc.SealSample(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sm)
}
