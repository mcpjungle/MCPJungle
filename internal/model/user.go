// Package model provides data models for the MCPJungle application.
package model

import (
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// User represents an authenticated, human user in enterprise mode.
// A user can be an admin or a regular user.
// There are no users if mcpjungle is running in development mode.
type User struct {
	gorm.Model

	Username    string         `json:"username" gorm:"unique; not null"`
	Role        types.UserRole `json:"role" gorm:"not null"`
	AccessToken string         `json:"access_token" gorm:"unique; not null"`

	// AllowList defines which MCP server names this user's self-created MCP clients
	// are allowed to access. Nil / empty means wildcard (all servers).
	// Stored as a JSON array of server names, e.g. ["atlassian", "github"].
	// Use "*" as a single entry to allow all servers explicitly.
	AllowList datatypes.JSON `json:"allow_list" gorm:"type:jsonb"`

	GroupID *uint  `json:"group_id" gorm:"index"`
	Group   *Group `json:"group,omitempty" gorm:"foreignKey:GroupID"`
}
