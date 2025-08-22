package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"log"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/audit"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// ListTools returns all tools registered in the registry.
func (m *MCPService) ListTools() ([]model.Tool, error) {
	var tools []model.Tool
	if err := m.db.Find(&tools).Error; err != nil {
		return nil, err
	}
	// prepend server name to tool names to ensure we only return the unique names of tools to user
	for i := range tools {
		var s model.McpServer
		if err := m.db.First(&s, "id = ?", tools[i].ServerID).Error; err != nil {
			return nil, fmt.Errorf("failed to get server for tool %s: %w", tools[i].Name, err)
		}
		tools[i].Name = mergeServerToolNames(s.Name, tools[i].Name)
	}
	return tools, nil
}

// ListToolsByServer fetches tools provided by an MCP server from the registry.
func (m *MCPService) ListToolsByServer(name string) ([]model.Tool, error) {
	if err := validateServerName(name); err != nil {
		return nil, err
	}

	s, err := m.GetMcpServer(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server %s from DB: %w", name, err)
	}

	var tools []model.Tool
	if err := m.db.Where("server_id = ?", s.ID).Find(&tools).Error; err != nil {
		return nil, fmt.Errorf("failed to get tools for server %s from DB: %w", name, err)
	}

	// prepend server name to tool names to ensure we only return the unique names of tools to user
	for i := range tools {
		tools[i].Name = mergeServerToolNames(s.Name, tools[i].Name)
	}

	return tools, nil
}

func (m *MCPService) GetTool(name string) (*model.Tool, error) {
	serverName, toolName, ok := splitServerToolName(name)
	if !ok {
		return nil, fmt.Errorf("invalid input: tool name does not contain a %s separator", serverToolNameSep)
	}

	s, err := m.GetMcpServer(serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server %s from DB: %w", serverName, err)
	}

	var tool model.Tool
	if err := m.db.Where("server_id = ? AND name = ?", s.ID, toolName).First(&tool).Error; err != nil {
		return nil, fmt.Errorf("failed to get tool %s from DB: %w", name, err)
	}
	// set the tool name back to its canonical form
	tool.Name = name
	return &tool, nil
}

// InvokeTool invokes a tool from a registered MCP server and returns its response.
func (m *MCPService) InvokeTool(ctx context.Context, name string, args map[string]any) (*types.ToolInvokeResult, error) {
	startTime := time.Now()

	// Generate unique request ID for this tool call
	requestID := uuid.New().String()

	serverName, toolName, ok := splitServerToolName(name)
	if !ok {
		return nil, fmt.Errorf("invalid input: tool name does not contain a %s separator", serverToolNameSep)
	}

	clientName := "mcpjungle"

	// Log the start of the tool call
	m.auditLogger.LogToolCallStart(ctx, audit.ToolCallStartEvent{
		RequestID:  requestID,
		ClientName: clientName,
		ServerName: serverName,
		ToolName:   toolName,
	})

	serverModel, err := m.GetMcpServer(serverName)
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
			"failed to get details about MCP server %s from DB: %w",
			serverName,
			err,
		)
	}

	mcpClient, err := newMcpServerSession(ctx, serverModel)
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

	callToolReq := mcp.CallToolRequest{}
	callToolReq.Params.Name = toolName
	callToolReq.Params.Arguments = args

	callToolResp, err := mcpClient.CallTool(ctx, callToolReq)

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
		return nil, fmt.Errorf("failed to call tool %s on MCP server %s: %w", toolName, serverName, err)
	}

	// NOTE: callToolResp.Content is a list of Content objects.
	// If the tool returns a list as its result, it gets converted to a list of Content objects.
	// But if the tool returns any other type of object (string, map, number, etc), then it is
	// completely available in Content[0].

	// Convert the Content field from []mcp.Content to []map[string]any to pass downstream.
	// We don't attempt to cast the data into specific types because this method should simply
	// forward the tool's response to the client.
	// It is up to the client of this API to convert the data into specific types like
	// Text, Image, etc.
	contentList := make([]map[string]any, 0, len(callToolResp.Content))
	for _, item := range callToolResp.Content {
		var m map[string]any
		serialized, err := json.Marshal(item)
		if err != nil {
			// TODO
			continue
		}
		if err = json.Unmarshal(serialized, &m); err != nil {
			// TODO
			continue
		}
		contentList = append(contentList, m)
	}

	result := &types.ToolInvokeResult{
		Meta:    callToolResp.Meta,
		IsError: callToolResp.IsError,
		Content: contentList,
	}

	return result, nil
}

// EnableTools enables one or more tools.
// If the entity is a tool name, only that tool is enabled.
// If the entity is a server name, all tools of that server are enabled.
// The function returns a list of enabled tool names.
// If the tool or server does not exist, it returns an error.
// If the tool is already enabled, it returns the tool name without an error.
func (m *MCPService) EnableTools(entity string) ([]string, error) {
	return m.setToolsEnabled(entity, true)
}

// DisableTools disables one or more tools.
// If the entity is a tool name, only that tool is disabled.
// If the entity is a server name, all tools of that server are disabled.
// The function returns a list of disabled tool names.
// If the tool or server does not exist, it returns an error.
// If the tool is already disabled, it returns the tool name without an error.
func (m *MCPService) DisableTools(entity string) ([]string, error) {
	return m.setToolsEnabled(entity, false)
}

// setToolsEnabled does the heavy lifting of enabling or disabling one or more tools.
func (m *MCPService) setToolsEnabled(entity string, enabled bool) ([]string, error) {
	serverName, toolName, ok := splitServerToolName(entity)
	if ok {
		// splitting was successful, so the entity is a tool name
		// only this tool needs to be enabled/disabled
		s, err := m.GetMcpServer(serverName)
		if err != nil {
			return nil, fmt.Errorf("failed to get MCP server %s: %w", serverName, err)
		}

		var tool model.Tool
		if err := m.db.Where("server_id = ? AND name = ?", s.ID, toolName).First(&tool).Error; err != nil {
			return nil, fmt.Errorf("failed to get tool %s: %w", entity, err)
		}

		if tool.Enabled == enabled {
			return []string{entity}, nil // no change needed
		}

		tool.Enabled = enabled
		if err := m.db.Save(&tool).Error; err != nil {
			return nil, fmt.Errorf("failed to set tool %s enabled=%t: %w", entity, enabled, err)
		}

		if enabled {
			// if the tool was enabled, add it back to the MCP proxy server
			mcpTool, err := convertToolModelToMcpObject(&tool)
			if err != nil {
				return nil, fmt.Errorf("failed to convert tool model to MCP object for tool %s: %w", tool.Name, err)
			}
			// set the tool name to its canonical form in the proxy
			mcpTool.Name = entity
			m.mcpProxyServer.AddTool(mcpTool, m.mcpProxyToolCallHandler)
		} else {
			// if the tool was disabled, remove it from the MCP proxy server
			m.mcpProxyServer.DeleteTools(entity)
		}

		return []string{entity}, nil
	}

	// splitting was unsuccessful, so the entity is a server name
	// all tools of this server need to be enabled/disabled
	s, err := m.GetMcpServer(entity)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server %s: %w", serverName, err)
	}

	var tools []model.Tool
	if err := m.db.Where("server_id = ?", s.ID).Find(&tools).Error; err != nil {
		return nil, fmt.Errorf("failed to get tools for server %s: %w", entity, err)
	}

	var changedToolNames []string
	for i := range tools {
		if tools[i].Enabled == enabled {
			continue // no change needed
		}
		tools[i].Enabled = enabled
		if err := m.db.Save(&tools[i]).Error; err != nil {
			return nil, fmt.Errorf("failed to set tool %s enabled=%t: %w", tools[i].Name, enabled, err)
		}
		canonicalToolName := mergeServerToolNames(s.Name, tools[i].Name)

		if enabled {
			mcpTool, err := convertToolModelToMcpObject(&tools[i])
			if err != nil {
				return nil, fmt.Errorf("failed to convert tool model to MCP object for tool %s: %w", tools[i].Name, err)
			}
			// set the tool name to its canonical form in the proxy
			mcpTool.Name = canonicalToolName

			m.mcpProxyServer.AddTool(mcpTool, m.mcpProxyToolCallHandler)
		} else {
			m.mcpProxyServer.DeleteTools(canonicalToolName)
		}

		changedToolNames = append(changedToolNames, canonicalToolName)
	}

	return changedToolNames, nil
}

// registerServerTools fetches all tools from an MCP server and registers them in the DB.
func (m *MCPService) registerServerTools(ctx context.Context, s *model.McpServer, c *client.Client) error {
	// fetch all tools from the server so they can be added to the DB
	resp, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("failed to fetch tools from MCP server %s: %w", s.Name, err)
	}
	for _, tool := range resp.Tools {
		canonicalToolName := mergeServerToolNames(s.Name, tool.GetName())

		// extracting json schema is currently on best-effort basis
		// if it fails, we log the error and continue with the next tool
		jsonSchema, _ := json.Marshal(tool.InputSchema)

		t := &model.Tool{
			ServerID:    s.ID,
			Name:        tool.GetName(),
			Description: tool.Description,
			InputSchema: jsonSchema,
		}
		if err := m.db.Create(t).Error; err != nil {
			// If registration of a tool fails, we should not fail the entire server registration.
			// Instead, continue with the next tool.
			log.Printf("[ERROR] failed to register tool %s in DB: %v", canonicalToolName, err)
		} else {
			// Set tool name to include the server name prefix to make it recognizable by MCPJungle
			// then add the tool to the MCP proxy server
			tool.Name = canonicalToolName
			m.mcpProxyServer.AddTool(tool, m.mcpProxyToolCallHandler)
		}
	}
	return nil
}

// deregisterServerTools deletes all tools that belong to an MCP server from the DB.
// It also removes the tools from the MCP proxy server.
func (m *MCPService) deregisterServerTools(s *model.McpServer) error {
	// load all tools for the server from the DB so we can delete them from the MCP proxy
	tools, err := m.ListToolsByServer(s.Name)
	if err != nil {
		return fmt.Errorf("failed to list tools for server %s: %w", s.Name, err)
	}

	// now it's safe to delete the server's tools from the DB
	result := m.db.Unscoped().Where("server_id = ?", s.ID).Delete(&model.Tool{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete tools for server %s: %w", s.Name, result.Error)
	}

	// delete tools from MCP proxy server
	toolNames := make([]string, len(tools), len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Name
	}
	m.mcpProxyServer.DeleteTools(toolNames...)

	return nil
}
