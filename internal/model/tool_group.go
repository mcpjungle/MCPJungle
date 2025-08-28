package model

import (
	"encoding/json"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ToolGroup represents a group of tools.
// It is useful when the user wants to expose only a subset of all tools to MCP clients.
type ToolGroup struct {
	gorm.Model

	Name        string `json:"name" gorm:"unique; not null"`
	Description string `json:"description"`

	// IncludedTools contains a list of tool names that are included in this group.
	// storing the list of tool names as a JSON array is a convenient way for now.
	IncludedTools datatypes.JSON `json:"included_tools" gorm:"type:jsonb; not null"`
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
