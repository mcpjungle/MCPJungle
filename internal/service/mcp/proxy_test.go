package mcp

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBForProxyAdditional(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.McpServer{},
		&model.Tool{},
		&model.Prompt{},
		&model.Resource{},
		&model.UpstreamOAuthToken{},
		&model.UpstreamOAuthPendingSession{},
	)
	require.NoError(t, err)

	return db
}

func newUpstreamStreamableHTTPServer(t *testing.T, upstream *mcpserver.MCPServer) *httptest.Server {
	t.Helper()
	return httptest.NewServer(mcpserver.NewStreamableHTTPServer(upstream))
}

func TestMCPProxyToolCallHandler_RewritesCanonicalNameAndForwardsArguments(t *testing.T) {
	db := setupTestDBForProxyAdditional(t)

	var seenToolName string
	var seenArgument string

	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0", mcpserver.WithToolCapabilities(true))
	upstream.AddTool(
		mcp.NewTool("echo", mcp.WithString("msg")),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			seenToolName = request.Params.Name
			if args, ok := request.GetArguments()["msg"].(string); ok {
				seenArgument = args
			}
			return mcp.NewToolResultText("upstream:" + seenArgument), nil
		},
	)

	httpServer := newUpstreamStreamableHTTPServer(t, upstream)
	defer httpServer.Close()

	srv := createStreamableHTTPTestServer(t, "tool-server", httpServer.URL)
	require.NoError(t, db.Create(srv).Error)

	service := &MCPService{
		db:                         db,
		metrics:                    telemetry.NewNoopCustomMetrics(),
		mcpServerInitReqTimeoutSec: 5,
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = "tool-server__echo"
	req.Params.Arguments = map[string]any{"msg": "hello"}

	res, err := service.MCPProxyToolCallHandler(context.WithValue(context.Background(), "mode", model.ModeDev), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "echo", seenToolName)
	assert.Equal(t, "hello", seenArgument)
}

func TestMCPProxyPromptHandler_RewritesCanonicalNameAndForwardsArguments(t *testing.T) {
	db := setupTestDBForProxyAdditional(t)

	var seenPromptName string
	var seenArgument string

	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0", mcpserver.WithPromptCapabilities(true))
	upstream.AddPrompt(
		mcp.NewPrompt("review", mcp.WithArgument("topic", mcp.RequiredArgument())),
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			seenPromptName = request.Params.Name
			seenArgument = request.Params.Arguments["topic"]
			return &mcp.GetPromptResult{
				Description: "Prompt result",
				Messages: []mcp.PromptMessage{
					mcp.NewPromptMessage(mcp.RoleAssistant, mcp.TextContent{Type: "text", Text: "ok"}),
				},
			}, nil
		},
	)

	httpServer := newUpstreamStreamableHTTPServer(t, upstream)
	defer httpServer.Close()

	srv := createStreamableHTTPTestServer(t, "prompt-server", httpServer.URL)
	require.NoError(t, db.Create(srv).Error)

	service := &MCPService{
		db:                         db,
		metrics:                    telemetry.NewNoopCustomMetrics(),
		mcpServerInitReqTimeoutSec: 5,
	}

	req := mcp.GetPromptRequest{}
	req.Params.Name = "prompt-server__review"
	req.Params.Arguments = map[string]string{"topic": "security"}

	res, err := service.mcpProxyPromptHandler(context.WithValue(context.Background(), "mode", model.ModeDev), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "review", seenPromptName)
	assert.Equal(t, "security", seenArgument)
}

func TestInitMCPProxyServer_LoadsEnabledEntitiesIntoCorrectProxyServers(t *testing.T) {
	db := setupTestDBForProxyAdditional(t)

	stdioServer := createTestServer(t, db)
	sseServer, err := model.NewSSEServer("sse-server", "SSE server", "https://example.com/sse", "", types.SessionModeStateless)
	require.NoError(t, err)
	require.NoError(t, db.Create(sseServer).Error)

	createTestToolRecord(t, db, stdioServer, "stdio-tool", true)
	createTestToolRecord(t, db, stdioServer, "disabled-tool", false)
	createTestPromptRecord(t, db, stdioServer, "stdio-prompt", true)
	createTestPromptRecord(t, db, stdioServer, "disabled-prompt", false)
	createTestResourceRecord(t, db, stdioServer, "resource://stdio/status", "stdio-status", true)
	createTestResourceRecord(t, db, stdioServer, "resource://stdio/disabled", "stdio-disabled", false)

	createTestToolRecord(t, db, sseServer, "sse-tool", true)
	createTestPromptRecord(t, db, sseServer, "sse-prompt", true)
	createTestResourceRecord(t, db, sseServer, "resource://sse/status", "sse-status", true)

	service := newTestLifecycleService(t, db)

	stdioTools := service.mcpProxyServer.ListTools()
	require.Contains(t, stdioTools, "test-server__stdio-tool")
	assert.NotContains(t, stdioTools, "test-server__disabled-tool")
	assert.NotContains(t, stdioTools, "sse-server__sse-tool")

	sseTools := service.sseMcpProxyServer.ListTools()
	require.Contains(t, sseTools, "sse-server__sse-tool")
	assert.NotContains(t, sseTools, "test-server__stdio-tool")

	stdioClient := newInitializedInProcessClient(t, service.mcpProxyServer)
	stdioPromptList, err := stdioClient.ListPrompts(context.Background(), mcp.ListPromptsRequest{})
	require.NoError(t, err)
	require.Len(t, stdioPromptList.Prompts, 1)
	assert.Equal(t, []string{"test-server__stdio-prompt"}, []string{stdioPromptList.Prompts[0].Name})

	stdioResourceList, err := stdioClient.ListResources(context.Background(), mcp.ListResourcesRequest{})
	require.NoError(t, err)
	require.Len(t, stdioResourceList.Resources, 1)
	assert.Equal(t, []string{buildResourceURI("test-server", "resource://stdio/status")}, []string{stdioResourceList.Resources[0].URI})

	sseClient := newInitializedInProcessClient(t, service.sseMcpProxyServer)
	ssePromptList, err := sseClient.ListPrompts(context.Background(), mcp.ListPromptsRequest{})
	require.NoError(t, err)
	require.Len(t, ssePromptList.Prompts, 1)
	assert.Equal(t, []string{"sse-server__sse-prompt"}, []string{ssePromptList.Prompts[0].Name})

	sseResourceList, err := sseClient.ListResources(context.Background(), mcp.ListResourcesRequest{})
	require.NoError(t, err)
	require.Len(t, sseResourceList.Resources, 1)
	assert.Equal(t, []string{buildResourceURI("sse-server", "resource://sse/status")}, []string{sseResourceList.Resources[0].URI})
}
