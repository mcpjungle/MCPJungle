package mcp

import (
	"context"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ProxyToolFilterMiddleware returns a mcp.Middleware that filters the tools/list response
// based on the request context (enterprise mode + client ACL).
func ProxyToolFilterMiddleware() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			result, err := next(ctx, method, req)
			if method != "tools/list" || err != nil {
				return result, err
			}
			listResult, ok := result.(*mcp.ListToolsResult)
			if !ok {
				return result, nil
			}
			listResult.Tools = proxyFilterTools(ctx, listResult.Tools)
			return listResult, nil
		}
	}
}

// proxyFilterTools filters tools based on enterprise mode and client allow-list from context.
func proxyFilterTools(ctx context.Context, tools []*mcp.Tool) []*mcp.Tool {
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

	var filteredTools []*mcp.Tool
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
