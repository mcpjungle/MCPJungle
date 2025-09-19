// Package config provides configuration service functionality for the MCPJungle application.
package config

import (
	"errors"
	"fmt"

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
