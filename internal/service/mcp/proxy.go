package mcp

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// initMCPProxyServer initializes the MCP proxy server.
// It loads all the registered MCP tools from the database into the proxy server.
func (m *MCPService) initMCPProxyServer() error {
	tools, err := m.ListTools()
	if err != nil {
		return fmt.Errorf("failed to list tools from DB: %w", err)
	}
	for _, tm := range tools {
		if !tm.Enabled {
			// do not add disabled tools to the proxy
			continue
		}

		// Add tool to the MCP proxy server
		tool, err := convertToolModelToMcpObject(&tm)
		if err != nil {
			return fmt.Errorf("failed to convert tool model to MCP object for tool %s: %w", tm.Name, err)
		}

		m.mcpProxyServer.AddTool(tool, m.mcpProxyToolCallHandler)
	}
	return nil
}

// mcpProxyToolCallHandler handles tool calls for the MCP proxy server
// by forwarding the request to the appropriate upstream MCP server and
// relaying the response back.
func (m *MCPService) mcpProxyToolCallHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := request.Params.Name
	serverName, toolName, ok := splitServerToolName(name)
	if !ok {
		return nil, fmt.Errorf("invalid input: tool name does not contain a %s separator", serverToolNameSep)
	}

	serverMode := ctx.Value("mode").(model.ServerMode)
	if serverMode == model.ModeProd {
		// In production mode, we need to check whether the MCP client is authorized to access the MCP server.
		// If not, return error Unauthorized.
		c := ctx.Value("client").(*model.McpClient)
		if !c.CheckHasServerAccess(serverName) {
			return nil, fmt.Errorf(
				"client %s is not authorized to access MCP server %s", c.Name, serverName,
			)
		}
	}

	// get the MCP server details from the database
	server, err := m.GetMcpServer(serverName)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get details about MCP server %s from DB: %w", serverName, err,
		)
	}

	var mcpClient *client.Client

	if server.Transport == types.TransportStreamableHTTP {

		mcpClient, err = createMcpServerConn(ctx, server)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create connection to MCP server %s: %w", serverName, err,
			)
		}
		defer mcpClient.Close()

	} else {

		// A new sub-process is spun up for each tool call to a STDIO mcp server.
		// This causes a serious performance hit, but is easy to implment so it is used for now.
		// TODO: Think of a better solution, ie, re-use connections to stdio MCP servers.
		mcpClient, err = runStdioServer(ctx, server)
		if err != nil {
			return nil, fmt.Errorf("failed to run stdio MCP server %s: %w", serverName, err)
		}
		defer mcpClient.Close()

	}

	// Ensure the tool name is set correctly, ie, without the server name prefix
	request.Params.Name = toolName

	// forward the request to the upstream MCP server and relay the response back
	return mcpClient.CallTool(ctx, request)
}
