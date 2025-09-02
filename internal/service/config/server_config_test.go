package config

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service"
)

func TestNewServerConfigService(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	svc := NewServerConfigService(db)
	service.AssertNotNil(t, svc)
	if svc.db != db {
		t.Errorf("Expected db to be %v, got %v", db, svc.db)
	}
}

func TestGetConfigEmptyDatabase(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the ServerConfig model
	err = db.AutoMigrate(&model.ServerConfig{})
	service.AssertNoError(t, err)

	svc := NewServerConfigService(db)

	config, err := svc.GetConfig()
	service.AssertNoError(t, err)

	// Should return default uninitialized config
	if config.Initialized {
		t.Error("Expected config to be uninitialized when database is empty")
	}
}

func TestGetConfigWithExistingConfig(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the ServerConfig model
	err = db.AutoMigrate(&model.ServerConfig{})
	service.AssertNoError(t, err)

	// Create a test config directly in the database
	testConfig := model.ServerConfig{
		Mode:        model.ModeDev,
		Initialized: true,
	}
	err = db.Create(&testConfig).Error
	service.AssertNoError(t, err)

	svc := NewServerConfigService(db)

	config, err := svc.GetConfig()
	service.AssertNoError(t, err)

	// Should return the existing config
	if !config.Initialized {
		t.Error("Expected config to be initialized")
	}
	if config.Mode != model.ModeDev {
		t.Errorf("Expected mode to be %v, got %v", model.ModeDev, config.Mode)
	}
}

func TestInitFirstTime(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the ServerConfig model
	err = db.AutoMigrate(&model.ServerConfig{})
	service.AssertNoError(t, err)

	svc := NewServerConfigService(db)

	// Initially no config should exist
	config, err := svc.GetConfig()
	service.AssertNoError(t, err)
	if config.Initialized {
		t.Error("Expected config to be uninitialized initially")
	}

	// Initialize the config
	created, err := svc.Init(model.ModeDev)
	service.AssertNoError(t, err)
	if !created {
		t.Error("Expected config to be created")
	}

	// Verify config was created
	config, err = svc.GetConfig()
	service.AssertNoError(t, err)
	if !config.Initialized {
		t.Error("Expected config to be initialized after Init")
	}
	if config.Mode != model.ModeDev {
		t.Errorf("Expected mode to be %v, got %v", model.ModeDev, config.Mode)
	}
}

func TestInitIdempotent(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the ServerConfig model
	err = db.AutoMigrate(&model.ServerConfig{})
	service.AssertNoError(t, err)

	svc := NewServerConfigService(db)

	// Initialize the config first time
	created, err := svc.Init(model.ModeDev)
	service.AssertNoError(t, err)
	if !created {
		t.Error("Expected config to be created first time")
	}

	// Try to initialize again
	created, err = svc.Init(model.ModeDev)
	service.AssertNoError(t, err)
	if created {
		t.Error("Expected config not to be created second time")
	}

	// Verify config is still valid
	config, err := svc.GetConfig()
	service.AssertNoError(t, err)
	if !config.Initialized {
		t.Error("Expected config to remain initialized")
	}
	if config.Mode != model.ModeDev {
		t.Errorf("Expected mode to remain %v, got %v", model.ModeDev, config.Mode)
	}
}

func TestInitWithDifferentMode(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the ServerConfig model
	err = db.AutoMigrate(&model.ServerConfig{})
	service.AssertNoError(t, err)

	svc := NewServerConfigService(db)

	// Initialize with dev mode
	created, err := svc.Init(model.ModeDev)
	service.AssertNoError(t, err)
	if !created {
		t.Error("Expected config to be created")
	}

	// Try to initialize with prod mode
	created, err = svc.Init(model.ModeProd)
	service.AssertNoError(t, err)
	if created {
		t.Error("Expected config not to be created when changing mode")
	}

	// Verify config mode was updated
	config, err := svc.GetConfig()
	service.AssertNoError(t, err)
	if config.Mode != model.ModeProd {
		t.Errorf("Expected mode to be updated to %v, got %v", model.ModeProd, config.Mode)
	}
}
