package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
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
				Port:          "8080",
				OtelProviders: nil, // Use nil for testing
				Metrics:       telemetry.NewNoopCustomMetrics(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.opts)
			if tt.wantErr {
				testhelpers.AssertError(t, err)
				// Check that server is nil when error occurs
				if server != nil {
					t.Error("Expected server to be nil when error occurs")
				}
			} else {
				testhelpers.AssertNoError(t, err)
				testhelpers.AssertNotNil(t, server)
				testhelpers.AssertEqual(t, tt.opts.Port, server.port)
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
		testhelpers.AssertError(t, err)
	})
}

func TestRouterSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	opts := &ServerOptions{
		Port: "8080",
	}

	server, err := NewServer(opts)
	testhelpers.AssertNoError(t, err)
	router, err := server.setupRouter()
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, router)

	// Test that health endpoint is registered
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)
	testhelpers.AssertEqual(t, http.StatusOK, w.Code)
}
