package model

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ToolResolver defines the interface needed to resolve tools by server.
type ToolResolver interface {
	// ListToolsByServer returns a list of tools for the given MCP server name.
	ListToolsByServer(serverName string) ([]Tool, error)
}

// ToolGroup represents a group of tools.
// It is useful when the user wants to expose only a subset of all tools to MCP clients.
type ToolGroup struct {
	gorm.Model

	Name        string `json:"name" gorm:"unique; not null"`
	Description string `json:"description"`

	// IncludedTools contains a list of tool names that are included in this group.
	// storing the list of tool names as a JSON array is a convenient way for now.
	IncludedTools datatypes.JSON `json:"included_tools" gorm:"type:jsonb"`

	// IncludedServers contains a list of MCP server names. All tools from these servers will be included.
	IncludedServers datatypes.JSON `json:"included_servers" gorm:"type:jsonb"`

	// ExcludedTools contains a list of tool names to exclude from the group.
	ExcludedTools datatypes.JSON `json:"excluded_tools" gorm:"type:jsonb"`
}

// GetTools unmarshals the IncludedTools JSON array into a slice of strings.
func (g *ToolGroup) GetTools() ([]string, error) {
	if g.IncludedTools == nil {
		return []string{}, nil
	}
	var tools []string
	err := json.Unmarshal(g.IncludedTools, &tools)
	return tools, err
}

// GetServers unmarshals the IncludedServers JSON array into a slice of strings.
func (g *ToolGroup) GetServers() ([]string, error) {
	if g.IncludedServers == nil {
		return []string{}, nil
	}
	var servers []string
	err := json.Unmarshal(g.IncludedServers, &servers)
	return servers, err
}

// GetExcludedTools unmarshals the ExcludedTools JSON array into a slice of strings.
func (g *ToolGroup) GetExcludedTools() ([]string, error) {
	if g.ExcludedTools == nil {
		return []string{}, nil
	}
	var tools []string
	err := json.Unmarshal(g.ExcludedTools, &tools)
	return tools, err
}

// ResolveEffectiveTools resolves all effective tools for this group by combining
// included_tools, included_servers, and applying excluded_tools.
// Note that tool exclusions are applied at last, so if a tool is both included and excluded,
// it will be excluded.
// It requires an MCP service to lookup tools by server.
func (g *ToolGroup) ResolveEffectiveTools(mcpService ToolResolver) ([]string, error) {
	effectiveTools := make(map[string]bool)

	// Add tools from included_tools
	includedTools, err := g.GetTools()
	if err != nil {
		return nil, fmt.Errorf("failed to get included tools: %w", err)
	}
	for _, tool := range includedTools {
		effectiveTools[tool] = true
	}

	// Add tools from included_servers
	includedServers, err := g.GetServers()
	if err != nil {
		return nil, fmt.Errorf("failed to get included servers: %w", err)
	}
	for _, serverName := range includedServers {
		serverTools, err := mcpService.ListToolsByServer(serverName)
		if err != nil {
			return nil, fmt.Errorf("failed to get tools for server %s: %w", serverName, err)
		}
		for _, tool := range serverTools {
			effectiveTools[tool.Name] = true
		}
	}

	// Remove tools from excluded_tools
	excludedTools, err := g.GetExcludedTools()
	if err != nil {
		return nil, fmt.Errorf("failed to get excluded tools: %w", err)
	}
	for _, tool := range excludedTools {
		delete(effectiveTools, tool)
	}

	// Convert map to slice
	result := make([]string, 0, len(effectiveTools))
	for tool := range effectiveTools {
		result = append(result, tool)
	}

	return result, nil
}
