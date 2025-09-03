package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Prompt represents a prompt template provided by an MCP server.
type Prompt struct {
	gorm.Model

	// Name is just the name of the prompt, without the server name prefix.
	// A prompt name is unique only within the context of a server.
	// This means that two prompts in mcpjungle DB CAN have the same name because
	// they belong to different servers, identified by server ID.
	Name string `json:"name" gorm:"not null"`

	// Enabled indicates whether the prompt is enabled or not.
	// If a prompt is disabled, it cannot be viewed or retrieved from the MCP proxy.
	Enabled bool `json:"enabled" gorm:"default:true"`

	Description string `json:"description"`

	// Arguments is a JSON schema that describes the input parameters for the prompt.
	Arguments datatypes.JSON `json:"arguments" gorm:"type:jsonb"`

	// ServerID is the ID of the MCP server that provides this prompt.
	ServerID uint      `json:"-" gorm:"not null"`
	Server   McpServer `json:"-" gorm:"foreignKey:ServerID;references:ID"`
}
