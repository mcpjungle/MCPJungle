// Package dashboardui embeds the compiled MCPJungle dashboard frontend and
// exposes an HTTP file server for serving those static assets from the main Go
// server. It does not build the frontend itself; it only serves the already
// generated bundle copied into internal/dashboardui/dist.
package dashboardui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist dist/*
var embeddedFiles embed.FS

// FileServer returns an http.Handler that serves the embedded dashboard asset
// bundle rooted at internal/dashboardui/dist.
func FileServer() (http.Handler, error) {
	subtree, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(subtree)), nil
}
