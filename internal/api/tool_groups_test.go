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

// setupToolGroupServer creates a Server with a real ToolGroupService backed by an in-memory DB.
func setupToolGroupServer(t *testing.T) *Server {
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

	return &Server{toolGroupService: tgSvc}
}

func TestGetToolGroupHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := setupToolGroupServer(t)

	router := gin.New()
	router.GET("/groups/:name", s.getToolGroupHandler())

	req := httptest.NewRequest(http.MethodGet, "/groups/ghost-group", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusNotFound, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "not found")
}

func TestUpdateToolGroupHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := setupToolGroupServer(t)

	router := gin.New()
	router.PUT("/groups/:name", s.updateToolGroupHandler())

	req := httptest.NewRequest(http.MethodPut, "/groups/ghost-group",
		strings.NewReader(`{"name":"ghost-group","description":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusNotFound, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "not found")
}

// setupToolGroupServerWithClients creates a Server backed by both a ToolGroupService and an
// McpClientService, with the cross-service bound-client checker wired in.
func setupToolGroupServerWithClients(t *testing.T) (*Server, *testhelpers.TestDBSetup) {
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
	tgSvc.SetMcpClientChecker(clientSvc)

	return &Server{toolGroupService: tgSvc, mcpClientService: clientSvc}, setup
}

// TestDeleteToolGroupHandler_ConflictWhenClientsBound verifies that deleting a Tool Group
// returns 409 Conflict when at least one MCP client is bound to it.
func TestDeleteToolGroupHandler_ConflictWhenClientsBound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, setup := setupToolGroupServerWithClients(t)

	// Seed a bound client directly in the DB (bypassing service validation for simplicity).
	groupName := "my-group"
	setup.CreateTestMcpClientBound("bound-client", "desc", "token-abc", nil, &groupName)

	router := gin.New()
	router.DELETE("/groups/:name", s.deleteToolGroupHandler())

	req := httptest.NewRequest(http.MethodDelete, "/groups/my-group", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusConflict, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "bound")
}

// TestDeleteToolGroupHandler_SucceedsWhenNoClientsBound verifies that deleting a Tool Group
// that has no bound clients succeeds (404 because the group doesn't exist in this test, but
// the 409 path is NOT triggered).
func TestDeleteToolGroupHandler_NoBoundClients_NoConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, _ := setupToolGroupServerWithClients(t)

	router := gin.New()
	router.DELETE("/groups/:name", s.deleteToolGroupHandler())

	// No clients bound — delete of a non-existent group should return 204 (idempotent), not 409.
	req := httptest.NewRequest(http.MethodDelete, "/groups/empty-group", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Must NOT be 409 Conflict.
	testhelpers.AssertTrue(t, w.Code != http.StatusConflict, "expected no 409 when no clients are bound")
}
