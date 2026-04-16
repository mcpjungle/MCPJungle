package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Resource represents a resource provided by an MCP server.
type Resource struct {
	gorm.Model

	// URI is the MCPJungle-assigned public URI for this resource.
	URI string `json:"uri" gorm:"not null"`

	// OriginalURI is the resource's original URI as advertised by the upstream MCP server
	OriginalURI string `json:"-" gorm:"not null"`

	// Name is the upstream display name of the resource, without the server name prefix.
	Name string `json:"name" gorm:"not null"`

	// Enabled indicates whether the resource is enabled or not.
	// If a resource is disabled, it cannot be viewed or read from the MCP proxy.
	Enabled bool `json:"enabled" gorm:"default:true"`

	Description string `json:"description"`
	MIMEType    string `json:"mime_type"`

	// Annotations stores upstream MCP resource annotations.
	Annotations datatypes.JSON `json:"annotations" gorm:"type:jsonb"`

	// Meta stores upstream MCP resource metadata.
	Meta datatypes.JSON `json:"meta" gorm:"type:jsonb"`

	// ServerID is the ID of the MCP server that provides this resource.
	ServerID uint      `json:"-" gorm:"not null"`
	Server   McpServer `json:"-" gorm:"foreignKey:ServerID;references:ID"`
}
