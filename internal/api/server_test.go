package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/service"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name    string
		opts    *ServerOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: &ServerOptions{
				Port: "8080",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.opts)
			if tt.wantErr {
				service.AssertError(t, err)
				// Check that server is nil when error occurs
				if server != nil {
					t.Error("Expected server to be nil when error occurs")
				}
			} else {
				service.AssertNoError(t, err)
				service.AssertNotNil(t, server)
				service.AssertEqual(t, tt.opts.Port, server.port)
			}
		})
	}
}

func TestServer_Start(t *testing.T) {
	// This test is limited due to the blocking nature of the Start method
	// We can only test that the method doesn't panic with invalid port
	t.Run("invalid port", func(t *testing.T) {
		// Create a minimal server with a router to avoid panic
		gin.SetMode(gin.TestMode)
		router := gin.New()

		server := &Server{
			port:   "invalid-port",
			router: router,
		}

		// This should return an error due to invalid port
		err := server.Start()
		service.AssertError(t, err)
	})
}

func TestRouterSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	opts := &ServerOptions{
		Port: "8080",
	}

	router, err := newRouter(opts)
	service.AssertNoError(t, err)
	service.AssertNotNil(t, router)

	// Test that health endpoint is registered
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)
	service.AssertEqual(t, http.StatusOK, w.Code)
}
