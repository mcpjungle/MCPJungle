// Package model provides data models for the MCPJungle application.
package model

import (
	"github.com/mcpjungle/mcpjungle/pkg/types"
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
}
