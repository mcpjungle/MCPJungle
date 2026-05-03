package ui

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func newDevProxy(rawURL string) (http.Handler, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		http.Error(w, "MCPJungle UI dev proxy error: "+err.Error(), http.StatusBadGateway)
	}

	return proxy, nil
}
