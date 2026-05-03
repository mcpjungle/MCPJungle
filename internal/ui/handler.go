package ui

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	uiBasePath = "/ui"
)

type Handler struct {
	proxy      http.Handler
	dist       fs.FS
	fileServer http.Handler
	hasIndex   bool
	indexHTML  []byte
}

func RegisterRoutes(r gin.IRouter) error {
	handler, err := NewHandlerFromEnv()
	if err != nil {
		return err
	}

	r.GET(uiBasePath, gin.WrapH(handler))
	r.GET(uiBasePath+"/*path", gin.WrapH(handler))

	return nil
}

func NewHandlerFromEnv() (*Handler, error) {
	if proxyURL := strings.TrimSpace(os.Getenv("MCPJUNGLE_UI_DEV_PROXY_URL")); proxyURL != "" {
		proxy, err := newDevProxy(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("parse MCPJUNGLE_UI_DEV_PROXY_URL: %w", err)
		}
		return &Handler{proxy: proxy}, nil
	}

	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("load embedded ui assets: %w", err)
	}

	_, err = fs.Stat(dist, "index.html")
	hasIndex := err == nil
	var indexHTML []byte
	if hasIndex {
		indexHTML, err = fs.ReadFile(dist, "index.html")
		if err != nil {
			return nil, fmt.Errorf("read embedded index.html: %w", err)
		}
	}

	return &Handler{
		dist:       dist,
		fileServer: http.StripPrefix(uiBasePath, http.FileServer(http.FS(dist))),
		hasIndex:   hasIndex,
		indexHTML:  indexHTML,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.proxy != nil {
		h.proxy.ServeHTTP(w, r)
		return
	}

	relPath := strings.TrimPrefix(r.URL.Path, uiBasePath)
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" {
		h.serveIndexOrUnavailable(w, r)
		return
	}

	cleanPath := path.Clean(relPath)
	if cleanPath == "." || cleanPath == "/" {
		h.serveIndexOrUnavailable(w, r)
		return
	}

	if fileExists(h.dist, cleanPath) {
		h.fileServer.ServeHTTP(w, r)
		return
	}

	if looksLikeAsset(cleanPath) {
		http.NotFound(w, r)
		return
	}

	h.serveIndexOrUnavailable(w, r)
}

func (h *Handler) serveIndexOrUnavailable(w http.ResponseWriter, r *http.Request) {
	if !h.hasIndex {
		http.Error(
			w,
			"MCPJungle UI assets not built. Run `npm run build` in `ui/` or set MCPJUNGLE_UI_DEV_PROXY_URL.",
			http.StatusServiceUnavailable,
		)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(h.indexHTML)
}

func fileExists(files fs.FS, name string) bool {
	info, err := fs.Stat(files, name)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func looksLikeAsset(name string) bool {
	if strings.HasPrefix(name, "assets/") {
		return true
	}

	switch path.Ext(name) {
	case ".css", ".js", ".json", ".ico", ".png", ".jpg", ".jpeg", ".svg", ".woff", ".woff2", ".map":
		return true
	default:
		return false
	}
}
