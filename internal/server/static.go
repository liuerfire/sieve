package server

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
)

//go:embed all:dist
var staticAssets embed.FS

func StaticHandler() http.Handler {
	dist, err := fs.Sub(staticAssets, "dist")
	if err != nil {
		// Log error and return a handler that serves a 500 error
		slog.Error("Failed to create sub filesystem for static assets", "err", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Internal Server Error: static assets not available", http.StatusInternalServerError)
		})
	}
	return http.FileServer(http.FS(dist))
}
