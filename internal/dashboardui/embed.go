package dashboardui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist dist/*
var embeddedFiles embed.FS

func FileServer() (http.Handler, error) {
	subtree, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(subtree)), nil
}
