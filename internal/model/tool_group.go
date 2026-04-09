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

// PromptResolver defines the interface needed to resolve prompts by server.
type PromptResolver interface {
	// ListPromptsByServer returns a list of prompts for the given MCP server name.
	ListPromptsByServer(serverName string) ([]Prompt, error)
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

	// IncludedTools contains a list of tool names that are included in this group.
	// storing the list of tool names as a JSON array is a convenient way for now.
	IncludedPrompts datatypes.JSON `json:"included_prompts" gorm:"type:jsonb"`

	// IncludedServers contains a list of MCP server names. All tools from these servers will be included.
	IncludedServers datatypes.JSON `json:"included_servers" gorm:"type:jsonb"`

	// ExcludedTools contains a list of tool names to exclude from the group.
	ExcludedTools datatypes.JSON `json:"excluded_tools" gorm:"type:jsonb"`

	// ExcludedPrompts contains a list of prompt names to exclude from the group.
	ExcludedPrompts datatypes.JSON `json:"excluded_prompts" gorm:"type:jsonb"`
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

// GetPrompts unmarshals the IncludedPrompts JSON array into a slice of strings.
func (g *ToolGroup) GetPrompts() ([]string, error) {
	if g.IncludedPrompts == nil {
		return []string{}, nil
	}
	var prompts []string
	err := json.Unmarshal(g.IncludedPrompts, &prompts)
	return prompts, err
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

// GetExcludedPrompts unmarshals the ExcludedPrompts JSON array into a slice of strings.
func (g *ToolGroup) GetExcludedPrompts() ([]string, error) {
	if g.ExcludedPrompts == nil {
		return []string{}, nil
	}
	var prompts []string
	err := json.Unmarshal(g.ExcludedPrompts, &prompts)
	return prompts, err
}

// resolveEffective is a helper that resolves effective items by combining
// included items, items from servers, and applying exclusions.
// It uses functional composition to work with both tools and prompts.
func (g *ToolGroup) resolveEffective(
	getIncluded func() ([]string, error),
	getExcluded func() ([]string, error),
	getFromServer func(serverName string) ([]string, error),
) ([]string, error) {
	effective := make(map[string]bool)

	// Add directly included items
	included, err := getIncluded()
	if err != nil {
		return nil, fmt.Errorf("failed to get included items: %w", err)
	}
	for _, item := range included {
		effective[item] = true
	}

	// Add items from included servers
	servers, err := g.GetServers()
	if err != nil {
		return nil, fmt.Errorf("failed to get included servers: %w", err)
	}
	for _, serverName := range servers {
		serverItems, err := getFromServer(serverName)
		if err != nil {
			return nil, fmt.Errorf("failed to get items for server %s: %w", serverName, err)
		}
		for _, item := range serverItems {
			effective[item] = true
		}
	}

	// Remove excluded items
	excluded, err := getExcluded()
	if err != nil {
		return nil, fmt.Errorf("failed to get excluded items: %w", err)
	}
	for _, item := range excluded {
		delete(effective, item)
	}

	// Convert map to slice
	result := make([]string, 0, len(effective))
	for item := range effective {
		result = append(result, item)
	}

	return result, nil
}

// ResolveEffectiveTools resolves all effective tools for this group by combining
// included_tools, included_servers, and applying excluded_tools.
// Note that exclusions are applied last, so if a tool is both included and excluded,
// it will be excluded.
func (g *ToolGroup) ResolveEffectiveTools(mcpService ToolResolver) ([]string, error) {
	return g.resolveEffective(
		g.GetTools,
		g.GetExcludedTools,
		g.resolveToolNames(mcpService),
	)
}

func (g *ToolGroup) resolveToolNames(mcpService ToolResolver) func(serverName string) ([]string, error) {
	return func(serverName string) ([]string, error) {
		tools, err := mcpService.ListToolsByServer(serverName)
		if err != nil {
			return nil, err
		}
		names := make([]string, len(tools))
		for i, t := range tools {
			names[i] = t.Name
		}
		return names, nil
	}
}

// ResolveEffectivePrompts resolves all effective prompts for this group by combining
// included_prompts, included_servers, and applying excluded_prompts.
// Note that exclusions are applied last, so if a prompt is both included and excluded,
// it will be excluded.
func (g *ToolGroup) ResolveEffectivePrompts(mcpService PromptResolver) ([]string, error) {
	return g.resolveEffective(
		g.GetPrompts,
		g.GetExcludedPrompts,
		g.resolvePromptNames(mcpService),
	)
}

func (g *ToolGroup) resolvePromptNames(mcpService PromptResolver) func(serverName string) ([]string, error) {
	return func(serverName string) ([]string, error) {
		prompts, err := mcpService.ListPromptsByServer(serverName)
		if err != nil {
			return nil, err
		}
		names := make([]string, len(prompts))
		for i, p := range prompts {
			names[i] = p.Name
		}
		return names, nil
	}
}
