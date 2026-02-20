package mcp

import (
	"context"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

// ProxyToolFilter filters tools exposed by MCP proxy for enterprise mode based on client allow-list.
func ProxyToolFilter(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
	serverMode := ctx.Value("mode").(model.ServerMode)
	if !model.IsEnterpriseMode(serverMode) {
		return tools
	}

	var filteredTools []mcp.Tool
	for _, tool := range tools {
		serverName, _, _ := splitServerToolName(tool.Name)
		c := ctx.Value("client").(*model.McpClient)
		if !c.CheckHasServerAccess(serverName) {
			log.Printf("[INFO] client %s is not authorized to access MCP tool %s", c.Name, tool.Name)
			continue
		}
		filteredTools = append(filteredTools, tool)
	}

	return filteredTools
}
