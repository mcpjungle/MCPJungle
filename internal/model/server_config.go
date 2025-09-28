package model

import (
	"fmt"

	"gorm.io/gorm"
)

type ServerMode string

const (
	// ModeDev is ideal for developers running the mcpjungle locally for personal MCP workflows.
	ModeDev ServerMode = "development"

	// ModeEnterprise is ideal for enterprise (production) deployments where multiple users will be using mcpjungle.
	ModeEnterprise ServerMode = "enterprise"

	// ModeProd is a deprecated alias for ModeEnterprise.
	// It exists for the sake of backward compatibility.
	ModeProd ServerMode = "production"
)

// IsEnterpriseMode returns true if the given server mode is an enterprise mode (ModeEnterprise or ModeProd),
// false otherwise.
// This function exists mainly for the sake of backward compatibility, since ModeProd is deprecated but still
// accepted as enterprise mode.
func IsEnterpriseMode(mode ServerMode) bool {
	return mode == ModeEnterprise || mode == ModeProd
}

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
	switch c.Mode {
	case ModeDev:
		// valid
	case ModeEnterprise:
		// valid
	case ModeProd:
		// valid but deprecated
		c.Mode = ModeEnterprise
	default:
		return fmt.Errorf("invalid server mode: %s", c.Mode)
	}
	return nil
}
