package model

import (
	"fmt"

	"gorm.io/gorm"
)

type ServerMode string

const (
	// ModeDev is ideal for individual developers running the server locally for
	// their personal MCP clients like Claude, Cursor, etc. and small use cases.
	ModeDev ServerMode = "development"

	// ModeEnterprise is ideal for enterprise (production) deployments
	ModeEnterprise ServerMode = "enterprise"
)

// ServerConfig represents the configuration for the MCPJungle server.
type ServerConfig struct {
	gorm.Model

	Mode ServerMode `gorm:"type:varchar(12);not null"`

	// Initialized indicates whether the server has been initialized.
	// If this is set to false, the server is not yet ready for use and all requests to it should be rejected.
	Initialized bool `gorm:"not null;default:false"`
}

func (c *ServerConfig) BeforeSave(tx *gorm.DB) (err error) {
	// Make sure that the server mode is valid before saving
	if c.Mode != ModeDev && c.Mode != ModeEnterprise {
		return fmt.Errorf("invalid server mode: %s", c.Mode)
	}
	return nil
}
