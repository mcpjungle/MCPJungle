package model

import "gorm.io/gorm"

// UserRole represents the role of a user in the MCPJungle system.
type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

// User represents a user in the MCPJungle system
type User struct {
	gorm.Model

	Username    string   `json:"username" gorm:"unique; not null"`
	Role        UserRole `json:"role" gorm:"not null"`
	AccessToken string   `json:"access_token" gorm:"unique; not null"`
}
