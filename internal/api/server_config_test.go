package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestRequireInitToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		secret         string
		remoteAddr     string
		headerValue    string
		expectedStatus int
	}{
		{
			name:           "no token configured, loopback allowed",
			secret:         "",
			remoteAddr:     "127.0.0.1:12345",
			headerValue:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no token configured, IPv6 loopback allowed",
			secret:         "",
			remoteAddr:     "[::1]:12345",
			headerValue:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no token configured, remote IP rejected",
			secret:         "",
			remoteAddr:     "203.0.113.5:12345",
			headerValue:    "",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "token configured, correct header allowed",
			secret:         "supersecret",
			remoteAddr:     "203.0.113.5:12345",
			headerValue:    "supersecret",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "token configured, wrong header rejected",
			secret:         "supersecret",
			remoteAddr:     "203.0.113.5:12345",
			headerValue:    "wrongtoken",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "token configured, missing header rejected",
			secret:         "supersecret",
			remoteAddr:     "203.0.113.5:12345",
			headerValue:    "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "token configured, loopback still needs header",
			secret:         "supersecret",
			remoteAddr:     "127.0.0.1:12345",
			headerValue:    "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/init", requireInitToken(tt.secret), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest(http.MethodPost, "/init", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.headerValue != "" {
				req.Header.Set("X-Init-Token", tt.headerValue)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			testhelpers.AssertEqual(t, tt.expectedStatus, w.Code)
		})
	}
}
