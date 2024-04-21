package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:build
var assets embed.FS

func FileServer() http.Handler {
	stripped, err := fs.Sub(assets, "build")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(stripped))
}
