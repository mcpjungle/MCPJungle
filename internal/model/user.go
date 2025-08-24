package model

import (
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// User represents a user in the MCPJungle system
type User struct {
	gorm.Model

	Username    string         `json:"username" gorm:"unique; not null"`
	Role        types.UserRole `json:"role" gorm:"not null"`
	AccessToken string         `json:"access_token" gorm:"unique; not null"`
}
