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

	session, err := m.getSession(ctx, server)
	if err != nil {
		outcome = telemetry.ToolCallOutcomeError
		return nil, err
	}
	defer session.closeIfApplicable()

	// Ensure the tool name is set correctly, ie, without the server name prefix
	request.Params.Name = toolName

	res, err := session.client.CallTool(ctx, request)
	if err != nil {
		outcome = telemetry.ToolCallOutcomeError
		session.invalidateOnError(err) // Invalidate unhealthy stateful sessions
	}

	// forward the request to the upstream MCP server and relay the response back
	return res, err
}

// mcpProxyPromptHandler handles prompt requests for the MCP proxy server
// by forwarding the request to the appropriate upstream MCP server and
// relaying the response back.
func (m *MCPService) mcpProxyPromptHandler(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	started := time.Now()
	outcome := telemetry.PromptCallOutcomeSuccess

	name := request.Params.Name
	serverName, promptName, ok := splitServerPromptName(name)
	if !ok {
		return nil, fmt.Errorf("invalid input: prompt name does not contain a %s separator", serverPromptNameSep)
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

	// Record the prompt call metrics at the end of the function
	defer func() {
		m.metrics.RecordPromptCall(ctx, serverName, promptName, outcome, time.Since(started))
	}()

	// get the MCP server details from the database
	server, err := m.GetMcpServer(serverName)
	if err != nil {
		// TODO: differentiate between "server not found" and other errors.
		// server not found is not an internal error, so outcome should be success.
		outcome = telemetry.PromptCallOutcomeError

		return nil, fmt.Errorf(
			"failed to get details about MCP server %s from DB: %w", serverName, err,
		)
	}

	session, err := m.getSession(ctx, server)
	if err != nil {
		outcome = telemetry.PromptCallOutcomeError
		return nil, err
	}
	defer session.closeIfApplicable()

	// Ensure the prompt name is set correctly, ie, without the server name prefix
	request.Params.Name = promptName

	// forward the request to the upstream MCP server and relay the response back
	res, err := session.client.GetPrompt(ctx, request)
	if err != nil {
		outcome = telemetry.PromptCallOutcomeError
		session.invalidateOnError(err) // Invalidate unhealthy stateful sessions
	}

	return res, err
}

// initMCPProxyServer initializes the MCP proxy server.
// It loads all the registered MCP tools and prompts from the database into the proxy server.
func (m *MCPService) initMCPProxyServer() error {
	mcpServerModelsCache := make(map[string]*model.McpServer)

	// Load Tools
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

	// Load prompts
	prompts, err := m.ListPrompts()
	if err != nil {
		return fmt.Errorf("failed to list prompts from DB: %w", err)
	}

	for _, pm := range prompts {
		if !pm.Enabled {
			// do not add disabled prompts to the proxy
			continue
		}

		// Add prompt to the MCP proxy server
		prompt, err := convertPromptModelToMcpObject(&pm)
		if err != nil {
			return fmt.Errorf("failed to convert prompt model to MCP object for prompt %s: %w", pm.Name, err)
		}

		// get the prompt's MCP server from cache so we can determine the transport type
		var server *model.McpServer
		serverName, _, _ := splitServerPromptName(prompt.Name)

		server, exists := mcpServerModelsCache[serverName]
		if !exists {
			server, err = m.GetMcpServer(serverName)
			if err != nil {
				return fmt.Errorf(
					"init mcp proxy server: failed to get MCP server %s for tool %s from DB: %w", serverName, prompt.Name, err,
				)
			}
			mcpServerModelsCache[serverName] = server
		}

		if server.Transport == types.TransportSSE {
			m.sseMcpProxyServer.AddPrompt(prompt, m.mcpProxyPromptHandler)
		} else {
			m.mcpProxyServer.AddPrompt(prompt, m.mcpProxyPromptHandler)
		}
	}

	return nil
}
