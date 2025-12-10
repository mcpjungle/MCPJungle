package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// ListPrompts returns all prompts registered in the registry.
func (m *MCPService) ListPrompts() ([]model.Prompt, error) {
	var prompts []model.Prompt
	if err := m.db.Find(&prompts).Error; err != nil {
		return nil, err
	}
	// prepend server name to prompt names to ensure we only return the unique names of prompts to user
	for i := range prompts {
		var s model.McpServer
		if err := m.db.First(&s, "id = ?", prompts[i].ServerID).Error; err != nil {
			return nil, fmt.Errorf("failed to get server for prompt %s: %w", prompts[i].Name, err)
		}
		prompts[i].Name = mergeServerPromptNames(s.Name, prompts[i].Name)
	}
	return prompts, nil
}

// ListPromptsByServer fetches prompts provided by an MCP server from the registry.
func (m *MCPService) ListPromptsByServer(name string) ([]model.Prompt, error) {
	if err := validateServerName(name); err != nil {
		return nil, err
	}

	s, err := m.GetMcpServer(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server %s from DB: %w", name, err)
	}

	var prompts []model.Prompt
	if err := m.db.Where("server_id = ?", s.ID).Find(&prompts).Error; err != nil {
		return nil, fmt.Errorf("failed to get prompts for server %s from DB: %w", name, err)
	}

	// prepend server name to prompt names to ensure we only return the unique names of prompts to user
	for i := range prompts {
		prompts[i].Name = mergeServerPromptNames(s.Name, prompts[i].Name)
	}

	return prompts, nil
}

// GetPrompt fetches a prompt from the database by its canonical name.
func (m *MCPService) GetPrompt(name string) (*model.Prompt, error) {
	serverName, promptName, ok := splitServerPromptName(name)
	if !ok {
		return nil, fmt.Errorf("invalid input: prompt name does not contain a %s separator", serverPromptNameSep)
	}

	s, err := m.GetMcpServer(serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server %s from DB: %w", serverName, err)
	}

	var prompt model.Prompt
	if err := m.db.Where("server_id = ? AND name = ?", s.ID, promptName).First(&prompt).Error; err != nil {
		return nil, fmt.Errorf("failed to get prompt %s from DB: %w", name, err)
	}
	// set the prompt name back to its canonical form
	prompt.Name = name
	return &prompt, nil
}

// GetPromptWithArgs retrieves a prompt with provided arguments and returns the rendered template.
func (m *MCPService) GetPromptWithArgs(ctx context.Context, name string, args map[string]any) (*types.PromptResult, error) {
	serverName, promptName, ok := splitServerPromptName(name)
	if !ok {
		return nil, fmt.Errorf("invalid input: prompt name does not contain a %s separator", serverPromptNameSep)
	}

	serverModel, err := m.GetMcpServer(serverName)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get details about MCP server %s from DB: %w",
			serverName,
			err,
		)
	}

	mcpClient, err := newMcpServerSession(ctx, serverModel, m.mcpServerInitReqTimeoutSec)
	if err != nil {
		return nil, err
	}
	defer mcpClient.Close()

	getPromptReq := mcp.GetPromptRequest{}
	getPromptReq.Params.Name = promptName

	// Convert map[string]any to map[string]string for MCP API
	stringArgs := make(map[string]string)
	for k, v := range args {
		if str, ok := v.(string); ok {
			stringArgs[k] = str
		} else {
			// Convert non-string values to JSON strings
			if jsonBytes, err := json.Marshal(v); err == nil {
				stringArgs[k] = string(jsonBytes)
			}
		}
	}
	getPromptReq.Params.Arguments = stringArgs

	getPromptResp, err := mcpClient.GetPrompt(ctx, getPromptReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt %s from MCP server %s: %w", promptName, serverName, err)
	}

	// Convert the response to our types.PromptResult format
	messages := make([]types.PromptMessage, len(getPromptResp.Messages))
	for i, msg := range getPromptResp.Messages {
		var content map[string]any
		serialized, err := json.Marshal(msg.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize prompt message content: %w", err)
		}
		if err = json.Unmarshal(serialized, &content); err != nil {
			return nil, fmt.Errorf("failed to deserialize prompt message content: %w", err)
		}

		messages[i] = types.PromptMessage{
			Role:    string(msg.Role),
			Content: content,
		}
	}

	metaMap := m.convertMCPMetaToMap(getPromptResp.Meta)

	result := &types.PromptResult{
		Description: getPromptResp.Description,
		Messages:    messages,
		Meta:        metaMap,
	}
	return result, nil
}

// EnablePrompts enables one or more prompts.
// If the entity is a prompt name, only that prompt is enabled.
// If the entity is a server name, all prompts of that server are enabled.
// The function returns a list of enabled prompt names.
func (m *MCPService) EnablePrompts(entity string) ([]string, error) {
	return m.setPromptsEnabled(entity, true)
}

// DisablePrompts disables one or more prompts.
// If the entity is a prompt name, only that prompt is disabled.
// If the entity is a server name, all prompts of that server are disabled.
// The function returns a list of disabled prompt names.
func (m *MCPService) DisablePrompts(entity string) ([]string, error) {
	return m.setPromptsEnabled(entity, false)
}

// setPromptsEnabled does the heavy lifting of enabling or disabling one or more prompts.
func (m *MCPService) setPromptsEnabled(entity string, enabled bool) ([]string, error) {
	serverName, promptName, ok := splitServerPromptName(entity)
	if ok {
		// splitting was successful, so the entity is a prompt name
		// only this prompt needs to be enabled/disabled
		s, err := m.GetMcpServer(serverName)
		if err != nil {
			return nil, fmt.Errorf("failed to get MCP server %s: %w", serverName, err)
		}

		var prompt model.Prompt
		if err := m.db.Where("server_id = ? AND name = ?", s.ID, promptName).First(&prompt).Error; err != nil {
			return nil, fmt.Errorf("failed to get prompt %s: %w", entity, err)
		}

		if prompt.Enabled == enabled {
			return []string{entity}, nil // no change needed
		}

		prompt.Enabled = enabled
		if err := m.db.Save(&prompt).Error; err != nil {
			return nil, fmt.Errorf("failed to set prompt %s enabled=%t: %w", entity, enabled, err)
		}

		if enabled {
			// if the prompt was enabled, add it back to the MCP proxy server
			mcpPrompt, err := convertPromptModelToMcpObject(&prompt)
			if err != nil {
				return nil, fmt.Errorf("failed to convert prompt model to MCP object for prompt %s: %w", prompt.Name, err)
			}
			// set the prompt name to its canonical form in the proxy
			mcpPrompt.Name = entity

			if s.Transport == types.TransportSSE {
				m.sseMcpProxyServer.AddPrompt(mcpPrompt, m.mcpProxyPromptHandler)
			} else {
				m.mcpProxyServer.AddPrompt(mcpPrompt, m.mcpProxyPromptHandler)
			}
		} else {
			// if the prompt was disabled, remove it from the MCP proxy server
			if s.Transport == types.TransportSSE {
				m.sseMcpProxyServer.DeletePrompts(entity)
			} else {
				m.mcpProxyServer.DeletePrompts(entity)
			}
		}

		return []string{entity}, nil
	}

	// splitting was unsuccessful, so the entity is a server name
	// all prompts of this server need to be enabled/disabled
	s, err := m.GetMcpServer(entity)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server %s: %w", serverName, err)
	}

	var prompts []model.Prompt
	if err := m.db.Where("server_id = ?", s.ID).Find(&prompts).Error; err != nil {
		return nil, fmt.Errorf("failed to get prompts for server %s: %w", entity, err)
	}

	var changedPromptNames []string
	for i := range prompts {
		if prompts[i].Enabled == enabled {
			continue // no change needed
		}
		prompts[i].Enabled = enabled
		if err := m.db.Save(&prompts[i]).Error; err != nil {
			return nil, fmt.Errorf("failed to set prompt %s enabled=%t: %w", prompts[i].Name, enabled, err)
		}
		canonicalPromptName := mergeServerPromptNames(s.Name, prompts[i].Name)

		if enabled {
			mcpPrompt, err := convertPromptModelToMcpObject(&prompts[i])
			if err != nil {
				return nil, fmt.Errorf("failed to convert prompt model to MCP object for prompt %s: %w", prompts[i].Name, err)
			}
			// set the prompt name to its canonical form in the proxy
			mcpPrompt.Name = canonicalPromptName

			if s.Transport == types.TransportSSE {
				m.sseMcpProxyServer.AddPrompt(mcpPrompt, m.mcpProxyPromptHandler)
			} else {
				m.mcpProxyServer.AddPrompt(mcpPrompt, m.mcpProxyPromptHandler)
			}
		} else {
			if s.Transport == types.TransportSSE {
				m.sseMcpProxyServer.DeletePrompts(canonicalPromptName)
			} else {
				m.mcpProxyServer.DeletePrompts(canonicalPromptName)
			}
		}

		changedPromptNames = append(changedPromptNames, canonicalPromptName)
	}

	return changedPromptNames, nil
}

// registerServerPrompts fetches all prompts from an MCP server and registers them in the DB.
func (m *MCPService) registerServerPrompts(ctx context.Context, s *model.McpServer, c *client.Client) error {
	// fetch all prompts from the server so they can be added to the DB
	resp, err := c.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		return fmt.Errorf("failed to fetch prompts from MCP server %s: %w", s.Name, err)
	}
	for _, prompt := range resp.Prompts {
		canonicalPromptName := mergeServerPromptNames(s.Name, prompt.GetName())

		// extracting json schema is currently on best-effort basis
		// if it fails, we log the error and continue with the next prompt
		jsonArguments, _ := json.Marshal(prompt.Arguments)

		p := &model.Prompt{
			ServerID:    s.ID,
			Name:        prompt.GetName(),
			Description: prompt.Description,
			Arguments:   jsonArguments,
		}
		if err := m.db.Create(p).Error; err != nil {
			// If registration of a prompt fails, we should not fail the entire server registration.
			// Instead, continue with the next prompt.
			log.Printf("[ERROR] failed to register prompt %s in DB: %v", canonicalPromptName, err)
		} else {
			// Set prompt name to include the server name prefix to make it recognizable by MCPJungle
			// then add the prompt to the MCP proxy server
			prompt.Name = canonicalPromptName

			if s.Transport == types.TransportSSE {
				m.sseMcpProxyServer.AddPrompt(prompt, m.mcpProxyPromptHandler)
			} else {
				m.mcpProxyServer.AddPrompt(prompt, m.mcpProxyPromptHandler)
			}
		}
	}
	return nil
}

// deregisterServerPrompts deletes all prompts that belong to an MCP server from the DB.
// It also removes the prompts from the MCP proxy server.
func (m *MCPService) deregisterServerPrompts(s *model.McpServer) error {
	// load all prompts for the server from the DB so we can delete them from the MCP proxy
	prompts, err := m.ListPromptsByServer(s.Name)
	if err != nil {
		return fmt.Errorf("failed to list prompts for server %s: %w", s.Name, err)
	}

	// now it's safe to delete the server's prompts from the DB
	result := m.db.Unscoped().Where("server_id = ?", s.ID).Delete(&model.Prompt{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete prompts for server %s: %w", s.Name, result.Error)
	}

	// delete prompts from MCP proxy server
	promptNames := make([]string, len(prompts))
	for i, prompt := range prompts {
		promptNames[i] = prompt.Name
	}

	if s.Transport == types.TransportSSE {
		m.sseMcpProxyServer.DeletePrompts(promptNames...)
	} else {
		m.mcpProxyServer.DeletePrompts(promptNames...)
	}

	return nil
}
