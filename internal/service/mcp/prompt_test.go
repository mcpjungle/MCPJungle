package mcp

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBWithPrompts(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.McpServer{}, &model.Tool{}, &model.Prompt{})
	require.NoError(t, err)

	return db
}

func createTestServer(t *testing.T, db *gorm.DB) *model.McpServer {
	server, err := model.NewStdioServer(
		"test-server",
		"Test MCP server",
		"echo",
		[]string{"hello"},
		nil,
	)
	require.NoError(t, err)

	err = db.Create(server).Error
	require.NoError(t, err)
	return server
}

func createTestPrompt(t *testing.T, db *gorm.DB, server *model.McpServer, name string) *model.Prompt {
	args := []mcp.PromptArgument{
		{
			Name:        "code",
			Description: "Code to review",
			Required:    true,
		},
	}
	argsJSON, _ := json.Marshal(args)

	prompt := &model.Prompt{
		Name:        name,
		Description: "Test prompt for code review",
		Arguments:   argsJSON,
		Enabled:     true,
		ServerID:    server.ID,
	}
	err := db.Create(prompt).Error
	require.NoError(t, err)
	return prompt
}

func TestListPrompts(t *testing.T) {
	db := setupTestDBWithPrompts(t)
	service := &MCPService{db: db}

	server := createTestServer(t, db)
	createTestPrompt(t, db, server, "code-review")
	createTestPrompt(t, db, server, "security-audit")

	prompts, err := service.ListPrompts()
	require.NoError(t, err)
	assert.Len(t, prompts, 2)

	// Check that server names are prepended
	expectedNames := []string{
		"test-server__code-review",
		"test-server__security-audit",
	}

	actualNames := []string{prompts[0].Name, prompts[1].Name}
	assert.ElementsMatch(t, expectedNames, actualNames)
}

func TestListPromptsByServer(t *testing.T) {
	db := setupTestDBWithPrompts(t)
	service := &MCPService{db: db}

	server := createTestServer(t, db)
	createTestPrompt(t, db, server, "code-review")

	prompts, err := service.ListPromptsByServer("test-server")
	require.NoError(t, err)
	assert.Len(t, prompts, 1)
	assert.Equal(t, "test-server__code-review", prompts[0].Name)
}

func TestGetPrompt(t *testing.T) {
	db := setupTestDBWithPrompts(t)
	service := &MCPService{db: db}

	server := createTestServer(t, db)
	originalPrompt := createTestPrompt(t, db, server, "code-review")

	prompt, err := service.GetPrompt("test-server__code-review")
	require.NoError(t, err)
	assert.Equal(t, "test-server__code-review", prompt.Name)
	assert.Equal(t, originalPrompt.Description, prompt.Description)
}

func TestGetPrompt_InvalidName(t *testing.T) {
	db := setupTestDBWithPrompts(t)
	service := &MCPService{db: db}

	// Test with invalid name (no separator)
	_, err := service.GetPrompt("invalid-name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not contain a __ separator")
}

func TestEnableDisablePrompts(t *testing.T) {
	db := setupTestDBWithPrompts(t)

	// Create a mock MCP proxy server for testing
	mcpProxyServer := server.NewMCPServer(
		"Test Proxy",
		"0.1.0",
		server.WithPromptCapabilities(true),
	)

	service := &MCPService{
		db:             db,
		mcpProxyServer: mcpProxyServer,
	}

	server := createTestServer(t, db)
	prompt := createTestPrompt(t, db, server, "code-review")

	// Test disable
	disabledPrompts, err := service.DisablePrompts("test-server__code-review")
	require.NoError(t, err)
	assert.Len(t, disabledPrompts, 1)
	assert.Equal(t, "test-server__code-review", disabledPrompts[0])

	// Verify prompt is disabled in DB
	var updatedPrompt model.Prompt
	err = db.First(&updatedPrompt, prompt.ID).Error
	require.NoError(t, err)
	assert.False(t, updatedPrompt.Enabled)

	// Test enable
	enabledPrompts, err := service.EnablePrompts("test-server__code-review")
	require.NoError(t, err)
	assert.Len(t, enabledPrompts, 1)
	assert.Equal(t, "test-server__code-review", enabledPrompts[0])

	// Verify prompt is enabled in DB
	err = db.First(&updatedPrompt, prompt.ID).Error
	require.NoError(t, err)
	assert.True(t, updatedPrompt.Enabled)
}

func TestEnableDisableServerPrompts(t *testing.T) {
	db := setupTestDBWithPrompts(t)

	// Create a mock MCP proxy server for testing
	mcpProxyServer := server.NewMCPServer(
		"Test Proxy",
		"0.1.0",
		server.WithPromptCapabilities(true),
	)

	service := &MCPService{
		db:             db,
		mcpProxyServer: mcpProxyServer,
	}

	server := createTestServer(t, db)
	createTestPrompt(t, db, server, "code-review")
	createTestPrompt(t, db, server, "security-audit")

	// Test disable all prompts for server
	disabledPrompts, err := service.DisablePrompts("test-server")
	require.NoError(t, err)
	assert.Len(t, disabledPrompts, 2)

	// Verify all prompts are disabled
	var prompts []model.Prompt
	err = db.Where("server_id = ?", server.ID).Find(&prompts).Error
	require.NoError(t, err)
	for _, prompt := range prompts {
		assert.False(t, prompt.Enabled)
	}

	// Test enable all prompts for server
	enabledPrompts, err := service.EnablePrompts("test-server")
	require.NoError(t, err)
	assert.Len(t, enabledPrompts, 2)

	// Verify all prompts are enabled
	err = db.Where("server_id = ?", server.ID).Find(&prompts).Error
	require.NoError(t, err)
	for _, prompt := range prompts {
		assert.True(t, prompt.Enabled)
	}
}

func TestMergeServerPromptNames(t *testing.T) {
	result := mergeServerPromptNames("github", "code-review")
	assert.Equal(t, "github__code-review", result)
}

func TestSplitServerPromptName(t *testing.T) {
	serverName, promptName, ok := splitServerPromptName("github__code-review")
	assert.True(t, ok)
	assert.Equal(t, "github", serverName)
	assert.Equal(t, "code-review", promptName)

	// Test invalid name
	_, _, ok = splitServerPromptName("invalid-name")
	assert.False(t, ok)
}

func TestConvertPromptModelToMcpObject(t *testing.T) {
	args := []mcp.PromptArgument{
		{
			Name:        "code",
			Description: "Code to review",
			Required:    true,
		},
	}
	argsJSON, _ := json.Marshal(args)

	promptModel := &model.Prompt{
		Name:        "code-review",
		Description: "Review code for issues",
		Arguments:   argsJSON,
	}

	mcpPrompt, err := convertPromptModelToMcpObject(promptModel)
	require.NoError(t, err)
	assert.Equal(t, "code-review", mcpPrompt.Name)
	assert.Equal(t, "Review code for issues", mcpPrompt.Description)
	assert.Len(t, mcpPrompt.Arguments, 1)
	assert.Equal(t, "code", mcpPrompt.Arguments[0].Name)
	assert.True(t, mcpPrompt.Arguments[0].Required)
}
