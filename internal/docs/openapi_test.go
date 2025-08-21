package docs

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGinPathToOpenAPIPath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/api/v0/servers/:name", "/api/v0/servers/{name}"},
		{"/docs/*any", "/docs/{any}"},
		{"/health", "/health"},
		{"/api/v1/tools/:server/:tool", "/api/v1/tools/{server}/{tool}"},
	}
	for _, tt := range tests {
		got := ginPathToOpenAPIPath(tt.in)
		assert.Equal(t, tt.want, got, "ginPathToOpenAPIPath(%q)", tt.in)
	}
}

func TestDeriveSummary(t *testing.T) {
	tests := []struct {
		in   gin.RouteInfo
		want string
	}{
		{gin.RouteInfo{Method: "GET", Handler: "pkg.Func"}, "Get Func"},
		{gin.RouteInfo{Method: "POST", Handler: "github.com/x/y.z/handler.func1"}, "Post func1"},
		{gin.RouteInfo{Method: "DELETE", Handler: "JustName"}, "Delete JustName"},
	}
	for _, tt := range tests {
		got := deriveSummary(tt.in)
		assert.Equal(t, tt.want, got, "deriveSummary(%+v)", tt.in)
	}
}

func TestDeriveTags(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"/api/v0/servers", []string{"servers"}},
		{"/servers", []string{"servers"}},
		{"/api/v1/clients/123", []string{"clients"}},
		{"/health", []string{"health"}},
		{"/", []string{""}},
	}
	for _, tt := range tests {
		got := deriveTags(tt.path)
		assert.Equal(t, tt.want, got, "deriveTags(%q)", tt.path)
	}
}
