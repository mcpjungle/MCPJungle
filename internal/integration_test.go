package internal

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/migrations"
	"github.com/mcpjungle/mcpjungle/internal/model"
	mcpService "github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPromptsIntegration(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = migrations.Migrate(db)
	require.NoError(t, err)

	// Create MCP proxy server with prompt capabilities
	mcpProxyServer := server.NewMCPServer(
		"Test MCPJungle Proxy",
		"0.0.1",
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
	)
	sseMcpProxyServer := server.NewMCPServer(
		"MCPJungle Proxy MCP Server for SSE transport",
		"0.0.1",
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
	)
	mcpMetrics := telemetry.NewNoopCustomMetrics()

	// Create MCP service
	conf := &mcpService.ServiceConfig{
		DB:                      db,
		McpProxyServer:          mcpProxyServer,
		SseMcpProxyServer:       sseMcpProxyServer,
		Metrics:                 mcpMetrics,
		McpServerInitReqTimeout: 10,
	}
	service, err := mcpService.NewMCPService(conf)
	require.NoError(t, err)

	// Create test server in database
	// Create a test MCP server with prompts
	testServer, err := model.NewStdioServer(
		"github",
		"GitHub MCP server",
		"npx",
		[]string{"-y", "@modelcontextprotocol/server-github"},
		map[string]string{},
		"",
	)
	require.NoError(t, err)
	err = db.Create(testServer).Error
	require.NoError(t, err)

	// Create test prompt
	args := []mcp.PromptArgument{
		{
			Name:        "code",
			Description: "Code to review",
			Required:    true,
		},
		{
			Name:        "language",
			Description: "Programming language",
			Required:    false,
		},
	}
	argsJSON, _ := json.Marshal(args)

	testPrompt := &model.Prompt{
		Name:        "code-review",
		Description: "Review code for security issues and best practices",
		Arguments:   argsJSON,
		Enabled:     true,
		ServerID:    testServer.ID,
	}
	err = db.Create(testPrompt).Error
	require.NoError(t, err)

	// Test listing prompts
	prompts, err := service.ListPrompts()
	require.NoError(t, err)
	assert.Len(t, prompts, 1)
	assert.Equal(t, "github__code-review", prompts[0].Name)

	// Test getting specific prompt
	prompt, err := service.GetPrompt("github__code-review")
	require.NoError(t, err)
	assert.Equal(t, "github__code-review", prompt.Name)
	assert.Equal(t, "Review code for security issues and best practices", prompt.Description)

	// Test enable/disable
	disabledPrompts, err := service.DisablePrompts("github__code-review")
	require.NoError(t, err)
	assert.Len(t, disabledPrompts, 1)

	enabledPrompts, err := service.EnablePrompts("github__code-review")
	require.NoError(t, err)
	assert.Len(t, enabledPrompts, 1)

	// Test listing by server
	serverPrompts, err := service.ListPromptsByServer("github")
	require.NoError(t, err)
	assert.Len(t, serverPrompts, 1)
	assert.Equal(t, "github__code-review", serverPrompts[0].Name)
}

// Note: Naming convention utilities are tested in the service package tests
