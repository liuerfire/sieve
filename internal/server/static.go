package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var staticAssets embed.FS

func StaticHandler() http.Handler {
	dist, _ := fs.Sub(staticAssets, "dist")
	return http.FileServer(http.FS(dist))
}
