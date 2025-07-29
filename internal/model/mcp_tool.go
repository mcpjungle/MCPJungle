package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Tool represents a tool provided by an MCP server.
type Tool struct {
	gorm.Model

	// Name is just the name of the tool, without the server name prefix.
	// A tool name is unique only within the context of a server.
	// This means that two tools in mcpjungle DB CAN have the same name because
	// they belong to different servers, identified by server ID.
	Name string `json:"name" gorm:"not null"`

	// Enabled indicates whether the tool is enabled or not.
	// If a tool is disabled, it cannot be viewed or called from the MCP proxy.
	Enabled bool `json:"enabled" gorm:"default:true"`

	Description string `json:"description"`

	// InputSchema is a JSON schema that describes the input parameters for the tool.
	InputSchema datatypes.JSON `json:"input_schema" gorm:"type:jsonb"`

	// ServerID is the ID of the MCP server that provides this tool.
	ServerID uint      `json:"-" gorm:"not null"`
	Server   McpServer `json:"-" gorm:"foreignKey:ServerID;references:ID"`
}
