package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/audit"
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
	startTime := time.Now()

	// Generate unique request ID for this tool call
	requestID := uuid.New().String()

	name := request.Params.Name
	serverName, toolName, ok := splitServerToolName(name)
	if !ok {
		return nil, fmt.Errorf("invalid input: tool name does not contain a %s separator", serverToolNameSep)
	}

	// Extract client information from context
	var clientName string
	serverMode := ctx.Value("mode").(model.ServerMode)
	if serverMode == model.ModeProd {
		// In production mode, we need to check whether the MCP client is authorized to access the MCP server.
		// If not, return error Unauthorized.
		c := ctx.Value("client").(*model.McpClient)
		clientName = c.Name
		if !c.CheckHasServerAccess(serverName) {
			// Log authorization failure
			m.auditLogger.LogToolCall(ctx, audit.ToolCallEvent{
				RequestID:    requestID,
				ClientName:   clientName,
				ServerName:   serverName,
				ToolName:     toolName,
				Success:      false,
				Duration:     time.Since(startTime),
				ErrorMessage: "client not authorized to access server",
			})
			return nil, fmt.Errorf(
				"client %s is not authorized to access MCP server %s", c.Name, serverName,
			)
		}
	} else {
		// In development mode, use a default client name
		clientName = "dev-client"
	}

	// Log the start of the tool call
	m.auditLogger.LogToolCallStart(ctx, audit.ToolCallStartEvent{
		RequestID:  requestID,
		ClientName: clientName,
		ServerName: serverName,
		ToolName:   toolName,
	})

	// get the MCP server details from the database
	server, err := m.GetMcpServer(serverName)
	if err != nil {
		m.auditLogger.LogToolCall(ctx, audit.ToolCallEvent{
			RequestID:    requestID,
			ClientName:   clientName,
			ServerName:   serverName,
			ToolName:     toolName,
			Success:      false,
			Duration:     time.Since(startTime),
			ErrorMessage: fmt.Sprintf("failed to get server details: %v", err),
		})
		return nil, fmt.Errorf(
			"failed to get details about MCP server %s from DB: %w", serverName, err,
		)
	}

	mcpClient, err := newMcpServerSession(ctx, server)
	if err != nil {
		m.auditLogger.LogToolCall(ctx, audit.ToolCallEvent{
			RequestID:    requestID,
			ClientName:   clientName,
			ServerName:   serverName,
			ToolName:     toolName,
			Success:      false,
			Duration:     time.Since(startTime),
			ErrorMessage: fmt.Sprintf("failed to create MCP session: %v", err),
		})
		return nil, err
	}
	defer mcpClient.Close()

	// Ensure the tool name is set correctly, ie, without the server name prefix
	request.Params.Name = toolName

	// forward the request to the upstream MCP server and relay the response back
	result, err := mcpClient.CallTool(ctx, request)

	// Log the completed tool call
	event := audit.ToolCallEvent{
		RequestID:  requestID,
		ClientName: clientName,
		ServerName: serverName,
		ToolName:   toolName,
		Success:    err == nil,
		Duration:   time.Since(startTime),
	}

	if err != nil {
		event.ErrorMessage = fmt.Sprintf("tool call failed: %v", err)
	}

	m.auditLogger.LogToolCall(ctx, event)

	if err != nil {
		return nil, err
	}

	return result, nil
}
