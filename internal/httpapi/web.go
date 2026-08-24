package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web/*
var webFiles embed.FS

// registerWebRoutes exposes the researcher-facing page alongside the JSON API.
// The page is embedded so the service remains a single-binary deployment.
func registerWebRoutes(mux *http.ServeMux) {
	content, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(content))
	mux.Handle("/", files)
}
