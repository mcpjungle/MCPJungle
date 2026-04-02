package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/service/mcpclient"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestUpdateMcpClientHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	s := &Server{mcpClientService: mcpclient.NewMCPClientService(setup.DB)}
	router := gin.New()
	router.PUT("/clients/:name", s.updateMcpClientHandler())

	req := httptest.NewRequest(http.MethodPut, "/clients/ghost-client",
		strings.NewReader(`{"access_token":"validtoken123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusNotFound, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "not found")
}

func TestUpdateMcpClientHandler_Exists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	setup.CreateTestMcpClient("my-client", "test client", "oldtoken123", nil)

	s := &Server{mcpClientService: mcpclient.NewMCPClientService(setup.DB)}
	router := gin.New()
	router.PUT("/clients/:name", s.updateMcpClientHandler())

	req := httptest.NewRequest(http.MethodPut, "/clients/my-client",
		strings.NewReader(`{"access_token":"newtoken456"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusOK, w.Code)
}
