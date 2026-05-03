package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
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

func TestUpdateMcpClientHandler_UpdatesFieldsAndRotatesToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	setup.CreateTestMcpClient("my-client", "old desc", "oldtoken123", []string{"server-a"})

	s := &Server{mcpClientService: mcpclient.NewMCPClientService(setup.DB)}
	router := gin.New()
	router.PUT("/clients/:name", s.updateMcpClientHandler())

	req := httptest.NewRequest(http.MethodPut, "/clients/my-client",
		strings.NewReader(`{"description":"new desc","allow_list":["*"],"rotate_access_token":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusOK, w.Code)

	var updated model.McpClient
	err := setup.DB.Where("name = ?", "my-client").First(&updated).Error
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertEqual(t, "new desc", updated.Description)
	testhelpers.AssertTrue(t, updated.AccessToken != "oldtoken123", "expected rotated access token")

	var allowList []string
	err = json.Unmarshal(updated.AllowList, &allowList)
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertEqual(t, "*", allowList[0])
}
