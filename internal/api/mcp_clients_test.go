package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	mcpserver "github.com/mark3labs/mcp-go/server"
	mcpSvc "github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/mcpclient"
	"github.com/mcpjungle/mcpjungle/internal/service/toolgroup"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

// setupMcpClientServer creates a Server wired with both McpClientService and ToolGroupService
// so bound_tool_group validation works end-to-end in tests.
func setupMcpClientServer(t *testing.T, requireBinding bool) (*Server, *testhelpers.TestDBSetup) {
	t.Helper()
	setup := testhelpers.SetupTestDB(t)
	t.Cleanup(setup.Cleanup)

	mcpProxy := mcpserver.NewMCPServer("test", "0.0.1")
	sseMcpProxy := mcpserver.NewMCPServer("test-sse", "0.0.1")
	svc, err := mcpSvc.NewMCPService(&mcpSvc.ServiceConfig{
		DB:                      setup.DB,
		McpProxyServer:          mcpProxy,
		SseMcpProxyServer:       sseMcpProxy,
		Metrics:                 telemetry.NewNoopCustomMetrics(),
		McpServerInitReqTimeout: 5,
	})
	if err != nil {
		t.Fatalf("failed to create MCP service: %v", err)
	}
	tgSvc, err := toolgroup.NewToolGroupService(setup.DB, svc)
	if err != nil {
		t.Fatalf("failed to create tool group service: %v", err)
	}

	clientSvc := mcpclient.NewMCPClientService(setup.DB)
	clientSvc.SetToolGroupService(tgSvc)

	s := &Server{
		mcpClientService:       clientSvc,
		requireClientTGBinding: requireBinding,
	}
	return s, setup
}

func TestCreateMcpClientHandler_StrictMode_RequiresBoundToolGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, _ := setupMcpClientServer(t, true /* strict */)

	router := gin.New()
	router.POST("/clients", s.createMcpClientHandler())

	req := httptest.NewRequest(http.MethodPost, "/clients",
		strings.NewReader(`{"name":"my-client"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusBadRequest, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "bound_tool_group")
}

func TestCreateMcpClientHandler_NonStrictMode_NoBoundToolGroupAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, _ := setupMcpClientServer(t, false /* non-strict */)

	router := gin.New()
	router.POST("/clients", s.createMcpClientHandler())

	req := httptest.NewRequest(http.MethodPost, "/clients",
		strings.NewReader(`{"name":"my-client"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Without strict mode, a client can be created without bound_tool_group.
	testhelpers.AssertEqual(t, http.StatusCreated, w.Code)
}

func TestCreateMcpClientHandler_NonExistentBoundToolGroup_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, _ := setupMcpClientServer(t, false)

	router := gin.New()
	router.POST("/clients", s.createMcpClientHandler())

	// Request a non-existent bound_tool_group — should fail with 400.
	req := httptest.NewRequest(http.MethodPost, "/clients",
		strings.NewReader(`{"name":"my-client","bound_tool_group":"ghost-group"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusBadRequest, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "ghost-group")
}

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
