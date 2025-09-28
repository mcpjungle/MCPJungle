package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// MCPProxyToolCallHandler handles tool calls for the MCP proxy server
// by forwarding the request to the appropriate upstream MCP server and
// relaying the response back.
func (m *MCPService) MCPProxyToolCallHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	started := time.Now()
	outcome := telemetry.ToolCallOutcomeSuccess

	name := request.Params.Name
	serverName, toolName, ok := splitServerToolName(name)
	if !ok {
		return nil, fmt.Errorf("invalid input: tool name does not contain a %s separator", serverToolNameSep)
	}

	serverMode := ctx.Value("mode").(model.ServerMode)
	if model.IsEnterpriseMode(serverMode) {
		// In enterprise mode, we need to check whether the MCP client is authorized to access the MCP server.
		// If not, return error Unauthorized.
		c := ctx.Value("client").(*model.McpClient)
		if !c.CheckHasServerAccess(serverName) {
			return nil, fmt.Errorf(
				"client %s is not authorized to access MCP server %s", c.Name, serverName,
			)
		}
	}

	// Record the tool call metrics at the end of the function
	defer func() {
		m.metrics.RecordToolCall(ctx, serverName, toolName, outcome, time.Since(started))
	}()

	// get the MCP server details from the database
	server, err := m.GetMcpServer(serverName)
	if err != nil {
		// TODO: differentiate between "server not found" and other errors.
		// server not found is not an internal error, so outcome should be success.
		outcome = telemetry.ToolCallOutcomeError

		return nil, fmt.Errorf(
			"failed to get details about MCP server %s from DB: %w", serverName, err,
		)
	}

	mcpClient, err := newMcpServerSession(ctx, server)
	if err != nil {
		outcome = telemetry.ToolCallOutcomeError
		return nil, err
	}
	defer mcpClient.Close()

	// Ensure the tool name is set correctly, ie, without the server name prefix
	request.Params.Name = toolName

	res, err := mcpClient.CallTool(ctx, request)
	if err != nil {
		outcome = telemetry.ToolCallOutcomeError
	}

	// forward the request to the upstream MCP server and relay the response back
	return res, err
}

// initMCPProxyServer initializes the MCP proxy server.
// It loads all the registered MCP tools from the database into the proxy server.
func (m *MCPService) initMCPProxyServer() error {
	mcpServerModelsCache := make(map[string]*model.McpServer)

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

		// get the tool's MCP server so we can determine the transport type
		// use a cache to avoid querying the DB multiple times for the same server
		// since multiple tools can belong to the same server
		var server *model.McpServer
		serverName, _, _ := splitServerToolName(tool.Name)

		server, exists := mcpServerModelsCache[serverName]
		if !exists {
			server, err = m.GetMcpServer(serverName)
			if err != nil {
				return fmt.Errorf(
					"init mcp proxy server: failed to get MCP server %s for tool %s from DB: %w", serverName, tool.Name, err,
				)
			}
			// store the server model in cache so we don't have to query the DB again for the same server
			mcpServerModelsCache[serverName] = server
		}

		if server.Transport == types.TransportSSE {
			m.sseMcpProxyServer.AddTool(tool, m.MCPProxyToolCallHandler)
		} else {
			m.mcpProxyServer.AddTool(tool, m.MCPProxyToolCallHandler)
		}

		m.addToolInstance(tool)
	}

	return nil
}
