package web

import (
	"embed"
	"io/fs"
)

//go:embed all:static all:*.html
var EmbeddedFiles embed.FS

// GetStaticFS returns the embedded static files filesystem
func GetStaticFS() (fs.FS, error) {
	return fs.Sub(EmbeddedFiles, "static")
}

// GetTemplateFS returns the embedded template files filesystem  
func GetTemplateFS() fs.FS {
	return EmbeddedFiles
}