package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

// ProxyToolFilter filters tools exposed by MCP proxy for enterprise mode based on client allow-list.
func ProxyToolFilter(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
	serverMode, ok := ctx.Value("mode").(model.ServerMode)
	if !ok {
		// Missing/invalid mode in context: fail closed.
		return nil
	}
	if !model.IsEnterpriseMode(serverMode) {
		// In non-enterprise mode, there are no access restrictions, so return all tools
		return tools
	}

	c, ok := ctx.Value("client").(*model.McpClient)
	if !ok || c == nil {
		// Enterprise mode requires authenticated client context; fail closed if absent.
		return nil
	}

	var filteredTools []mcp.Tool
	allowedServers := make(map[string]bool)

	for _, tool := range tools {
		serverName, _, _ := splitServerToolName(tool.Name)

		allowed, cached := allowedServers[serverName]
		if !cached {
			// check whether the client has access to this server and cache the result for faster future checks
			allowed = c.CheckHasServerAccess(serverName)
			allowedServers[serverName] = allowed
		}
		if allowed {
			// client has access to this tool's server, so include it in the filtered list
			filteredTools = append(filteredTools, tool)
		}
	}

	return filteredTools
}
