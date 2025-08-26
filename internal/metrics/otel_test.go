package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestProviders_RegisterMetricsHandlers(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected int // expected number of handlers registered
	}{
		{
			name:     "disabled",
			enabled:  false,
			expected: 0,
		},
		{
			name:     "enabled",
			enabled:  true,
			expected: 2, // metrics and health endpoints
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Enabled:     tt.enabled,
				MetricsPath: "/metrics",
			}

			var providers *Providers
			if tt.enabled {
				// Create a proper Providers instance with Prometheus handler
				providers = &Providers{
					Config:  config,
					Handler: promhttp.Handler(),
				}
			} else {
				providers = &Providers{
					Config: config,
				}
			}

			mux := http.NewServeMux()
			providers.RegisterMetricsHandlers(mux)

			// Count registered handlers by checking if endpoints respond
			handlerCount := 0
			if tt.enabled {
				// Test metrics endpoint
				req, _ := http.NewRequest("GET", "/metrics", nil)
				rr := httptest.NewRecorder()
				mux.ServeHTTP(rr, req)
				// Metrics endpoint should return 200 OK with Prometheus metrics
				if rr.Code == http.StatusOK {
					handlerCount++
				}

				// Test health endpoint
				req, _ = http.NewRequest("GET", "/health", nil)
				rr = httptest.NewRecorder()
				mux.ServeHTTP(rr, req)
				if rr.Code == http.StatusOK {
					handlerCount++
				}
			}

			if handlerCount != tt.expected {
				t.Errorf("Expected %d handlers to be registered, got %d", tt.expected, handlerCount)
			}
		})
	}
}
