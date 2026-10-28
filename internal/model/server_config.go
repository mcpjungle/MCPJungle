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

	// UpstreamTLS holds TLS trust configuration for MCPJungle-to-upstream MCP server connections.
	UpstreamTLS UpstreamTLSServerConfig `gorm:"embedded;embeddedPrefix:upstream_tls_"`
}

// UpstreamTLSServerConfig holds TLS trust configuration for MCPJungle-to-upstream MCP server connections.
// This configuration applies exclusively to connections initiated by MCPJungle toward upstream MCP servers,
// not to client-to-gateway TLS termination.
type UpstreamTLSServerConfig struct {
	// TLSCAFile is the path to a CA certificate file used to verify upstream MCP server TLS certificates.
	// When empty, the system's default CA certificates are used.
	TLSCAFile string `gorm:"type:varchar(500)"`

	// TLSInsecureSkipVerify controls whether a client verifies the upstream MCP server's
	// certificate chain and host name. If TLSInsecureSkipVerify is true, TLS accepts any
	// certificate presented by the server and any host name in that certificate.
	// This should only be used for local/development workflows.
	TLSInsecureSkipVerify bool `gorm:"default:false"`
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
