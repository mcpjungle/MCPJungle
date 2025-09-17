package config

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestNewServerConfigService(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	svc := NewServerConfigService(db)
	testhelpers.AssertNotNil(t, svc)
	if svc.db != db {
		t.Errorf("Expected db to be %v, got %v", db, svc.db)
	}
}

func TestGetConfigEmptyDatabase(t *testing.T) {
	setup := testhelpers.SetupServerConfigTest(t)
	defer setup.Cleanup()

	svc := NewServerConfigService(setup.DB)

	config, err := svc.GetConfig()
	testhelpers.AssertNoError(t, err)

	// Should return default uninitialized config
	if config.Initialized {
		t.Error("Expected config to be uninitialized when database is empty")
	}
}

func TestGetConfigWithExistingConfig(t *testing.T) {
	setup := testhelpers.SetupServerConfigTest(t)
	defer setup.Cleanup()

	// Create a test config using the helper
	setup.CreateTestServerConfig(model.ModeDev, true)

	svc := NewServerConfigService(setup.DB)

	config, err := svc.GetConfig()
	testhelpers.AssertNoError(t, err)

	// Should return the existing config
	if !config.Initialized {
		t.Error("Expected config to be initialized")
	}
	if config.Mode != model.ModeDev {
		t.Errorf("Expected mode to be %v, got %v", model.ModeDev, config.Mode)
	}
}

func TestInitFirstTime(t *testing.T) {
	setup := testhelpers.SetupServerConfigTest(t)
	defer setup.Cleanup()

	svc := NewServerConfigService(setup.DB)

	// Initially no config should exist
	config, err := svc.GetConfig()
	testhelpers.AssertNoError(t, err)
	if config.Initialized {
		t.Error("Expected config to be uninitialized initially")
	}

	// Initialize the config
	created, err := svc.Init(model.ModeDev)
	testhelpers.AssertNoError(t, err)
	if !created {
		t.Error("Expected config to be created")
	}

	// Verify config was created
	config, err = svc.GetConfig()
	testhelpers.AssertNoError(t, err)
	if !config.Initialized {
		t.Error("Expected config to be initialized after Init")
	}
	if config.Mode != model.ModeDev {
		t.Errorf("Expected mode to be %v, got %v", model.ModeDev, config.Mode)
	}
}

func TestInitIdempotent(t *testing.T) {
	db, err := testhelpers.CreateTestDB()
	testhelpers.AssertNoError(t, err)

	// Auto-migrate the ServerConfig model
	err = db.AutoMigrate(&model.ServerConfig{})
	testhelpers.AssertNoError(t, err)

	svc := NewServerConfigService(db)

	// Initialize the config first time
	created, err := svc.Init(model.ModeDev)
	testhelpers.AssertNoError(t, err)
	if !created {
		t.Error("Expected config to be created first time")
	}

	// Try to initialize again
	created, err = svc.Init(model.ModeDev)
	testhelpers.AssertNoError(t, err)
	if created {
		t.Error("Expected config not to be created second time")
	}

	// Verify config is still valid
	config, err := svc.GetConfig()
	testhelpers.AssertNoError(t, err)
	if !config.Initialized {
		t.Error("Expected config to remain initialized")
	}
	if config.Mode != model.ModeDev {
		t.Errorf("Expected mode to remain %v, got %v", model.ModeDev, config.Mode)
	}
}
