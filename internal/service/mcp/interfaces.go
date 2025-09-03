package mcp

import (
	"context"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// PromptManager defines operations for managing MCP prompts
// Following Interface Segregation Principle - focused on prompt operations only
type PromptManager interface {
	ListPrompts() ([]model.Prompt, error)
	ListPromptsByServer(serverName string) ([]model.Prompt, error)
	GetPrompt(name string) (*model.Prompt, error)
	GetPromptWithArgs(ctx context.Context, name string, args map[string]any) (*types.PromptResult, error)
	EnablePrompts(names ...string) ([]string, error)
	DisablePrompts(names ...string) ([]string, error)
}

// ToolManager defines operations for managing MCP tools
// Extracted from existing implementation for better separation
type ToolManager interface {
	ListTools() ([]model.Tool, error)
	ListToolsByServer(serverName string) ([]model.Tool, error)
	GetTool(name string) (*model.Tool, error)
	InvokeTool(ctx context.Context, name string, args map[string]any) (*types.ToolInvokeResult, error)
	EnableTools(names ...string) ([]string, error)
	DisableTools(names ...string) ([]string, error)
}

// ServerManager defines operations for managing MCP servers
type ServerManager interface {
	RegisterMcpServer(ctx context.Context, s *model.McpServer) error
	DeregisterMcpServer(name string) error
	ListMcpServers() ([]model.McpServer, error)
	GetMcpServer(name string) (*model.McpServer, error)
}

// MCPServiceInterface combines all MCP operations
// Following Dependency Inversion Principle - depend on abstraction
type MCPServiceInterface interface {
	PromptManager
	ToolManager
	ServerManager
}
