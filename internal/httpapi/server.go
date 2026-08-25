// Package httpapi 暴露 HTTP 接口（路由前缀 /api）。仅调用 service 层，不直接碰 store。
// 使用 Go 1.22+ 增强型 ServeMux（方法 + 路径通配）。
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"task215-fiberlineage/internal/model"
	"task215-fiberlineage/internal/service"
)

// Server 持有 service，提供 HTTP 路由。
type Server struct {
	svc *service.Service
}

// NewServer 构建 HTTP Server。
func NewServer(svc *service.Service) *Server { return &Server{svc: svc} }

// Handler 返回已注册路由的 http.Handler。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	registerWebRoutes(mux)
	s.registerSampleRoutes(mux)
	s.registerEvidenceRoutes(mux)
	s.registerCompareRoutes(mux)
	s.registerLineageRoutes(mux)
	s.registerVersionRoutes(mux)
	s.registerMiscRoutes(mux)
	return mux
}

// writeJSON 写出 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 将领域错误映射为 HTTP 状态码并写出。
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case model.IsNotFound(err):
		status = http.StatusNotFound
	case model.IsInvalidInput(err):
		status = http.StatusBadRequest
	case model.IsInvalidState(err), model.IsConflict(err), model.IsDuplicate(err):
		status = http.StatusConflict
	case model.IsForbidden(err):
		status = http.StatusForbidden
	}
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

// decodeBody 解析请求体为 v。
func decodeBody(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}
