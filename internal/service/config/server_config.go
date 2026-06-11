// Package config provides configuration service functionality for the MCPJungle application.
package config

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
)

// ServerConfigService provides methods to manage server configuration in the database.
type ServerConfigService struct {
	db *gorm.DB
}

func NewServerConfigService(db *gorm.DB) *ServerConfigService {
	return &ServerConfigService{db: db}
}

// GetConfig retrieves the server configuration from the database.
// If no configuration exists, it returns a default uninitialized config.
func (s *ServerConfigService) GetConfig() (model.ServerConfig, error) {
	var config model.ServerConfig
	err := s.db.First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.ServerConfig{Initialized: false}, nil
	}
	if err != nil {
		return model.ServerConfig{}, fmt.Errorf("failed to fetch server configuration from db: %v", err)
	}
	if config.UpstreamTLS.TLSInsecureSkipVerify {
		log.Printf("[WARN] TLS certificate verification is disabled for upstream MCP server connections (TLSInsecureSkipVerify=true). This should only be used for local/development workflows and must not be enabled in production.")
	}
	if config.UpstreamTLS.TLSCAFile != "" {
		if _, err := os.Stat(config.UpstreamTLS.TLSCAFile); err != nil {
			return model.ServerConfig{}, fmt.Errorf("upstream TLS CA file does not exist: %s", config.UpstreamTLS.TLSCAFile)
		}
	}
	return config, nil
}

// Init initializes the server configuration in the database.
// It is an idempotent operation. It returns true if the config was created.
// If the config already exists, it returns false and does nothing else.
func (s *ServerConfigService) Init(mode model.ServerMode) (bool, error) {
	config, err := s.GetConfig()
	if err != nil {
		return false, err
	}
	if config.Initialized {
		// Config already exists, do nothing
		return false, nil
	}
	// No config exists, create one
	config = model.ServerConfig{
		Mode:        mode,
		Initialized: true,
	}
	return true, s.db.Create(&config).Error
}
