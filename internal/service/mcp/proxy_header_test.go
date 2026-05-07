package mcp

import (
	"context"
	"net/http"
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

func setupProxyHeaderTestDB(t *testing.T) *gorm.DB {
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

func createStreamableHTTPTestServer(t *testing.T, dbName, upstreamURL string) *model.McpServer {
	t.Helper()

	srv, err := model.NewStreamableHTTPServer(
		dbName,
		"Test streamable HTTP server",
		upstreamURL,
		"",
		nil,
		types.SessionModeStateless,
	)
	require.NoError(t, err)

	return srv
}

func TestMCPProxyToolCallHandlerStripsInboundHeaders(t *testing.T) {
	db := setupProxyHeaderTestDB(t)

	var seenAuthorization string
	var seenCustomHeader string

	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0")
	upstream.AddTool(
		mcp.NewTool("echo", mcp.WithString("msg")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			seenAuthorization = req.Header.Get("Authorization")
			seenCustomHeader = req.Header.Get("X-Test-Forward")
			msg, _ := req.GetArguments()["msg"].(string)
			return mcp.NewToolResultText(msg), nil
		},
	)

	httpServer := httptest.NewServer(mcpserver.NewStreamableHTTPServer(upstream))
	defer httpServer.Close()

	srv := createStreamableHTTPTestServer(t, "test-server", httpServer.URL)
	require.NoError(t, db.Create(srv).Error)

	service := &MCPService{
		db:                         db,
		metrics:                    telemetry.NewNoopCustomMetrics(),
		mcpServerInitReqTimeoutSec: 5,
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = "test-server__echo"
	req.Params.Arguments = map[string]any{"msg": "hello"}
	req.Header = http.Header{
		"Authorization":   []string{"Bearer downstream-token"},
		"X-Test-Forward":  []string{"should-not-forward"},
		"Accept-Encoding": []string{"gzip"},
	}

	res, err := service.MCPProxyToolCallHandler(context.WithValue(context.Background(), "mode", model.ModeDev), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, seenAuthorization)
	assert.Empty(t, seenCustomHeader)
}

func TestMCPProxyPromptHandlerStripsInboundHeaders(t *testing.T) {
	db := setupProxyHeaderTestDB(t)

	var seenAuthorization string
	var seenCustomHeader string

	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0")
	upstream.AddPrompt(
		mcp.Prompt{Name: "review"},
		func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			seenAuthorization = req.Header.Get("Authorization")
			seenCustomHeader = req.Header.Get("X-Test-Forward")
			return &mcp.GetPromptResult{
				Messages: []mcp.PromptMessage{
					mcp.NewPromptMessage(mcp.RoleAssistant, mcp.TextContent{Type: "text", Text: "ok"}),
				},
			}, nil
		},
	)

	httpServer := httptest.NewServer(mcpserver.NewStreamableHTTPServer(upstream))
	defer httpServer.Close()

	srv := createStreamableHTTPTestServer(t, "test-server", httpServer.URL)
	require.NoError(t, db.Create(srv).Error)

	service := &MCPService{
		db:                         db,
		metrics:                    telemetry.NewNoopCustomMetrics(),
		mcpServerInitReqTimeoutSec: 5,
	}

	req := mcp.GetPromptRequest{}
	req.Params.Name = "test-server__review"
	req.Header = http.Header{
		"Authorization":  []string{"Bearer downstream-token"},
		"X-Test-Forward": []string{"should-not-forward"},
	}

	res, err := service.mcpProxyPromptHandler(context.WithValue(context.Background(), "mode", model.ModeDev), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, seenAuthorization)
	assert.Empty(t, seenCustomHeader)
}

func TestMCPProxyResourceHandlerStripsInboundHeaders(t *testing.T) {
	db := setupProxyHeaderTestDB(t)

	var seenAuthorization string
	var seenCustomHeader string

	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0")
	upstream.AddResource(
		mcp.Resource{
			URI:         "resource://test/status",
			Name:        "status",
			Description: "Current status",
			MIMEType:    "text/plain",
		},
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			seenAuthorization = req.Header.Get("Authorization")
			seenCustomHeader = req.Header.Get("X-Test-Forward")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      req.Params.URI,
					MIMEType: "text/plain",
					Text:     "ok",
				},
			}, nil
		},
	)

	httpServer := httptest.NewServer(mcpserver.NewStreamableHTTPServer(upstream))
	defer httpServer.Close()

	srv := createStreamableHTTPTestServer(t, "test-server", httpServer.URL)
	require.NoError(t, db.Create(srv).Error)
	createTestResource(t, db, srv, "resource://test/status", "status")

	service := &MCPService{
		db:                         db,
		metrics:                    telemetry.NewNoopCustomMetrics(),
		mcpServerInitReqTimeoutSec: 5,
	}

	req := mcp.ReadResourceRequest{}
	req.Params.URI = buildResourceURI("test-server", "resource://test/status")
	req.Header = http.Header{
		"Authorization":  []string{"Bearer downstream-token"},
		"X-Test-Forward": []string{"should-not-forward"},
	}

	res, err := service.mcpProxyResourceHandler(context.WithValue(context.Background(), "mode", model.ModeDev), req)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Empty(t, seenAuthorization)
	assert.Empty(t, seenCustomHeader)
}
