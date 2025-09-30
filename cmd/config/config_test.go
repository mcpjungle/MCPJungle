package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestClientConfig(t *testing.T) {
	t.Run("ClientConfig struct has AccessToken field", func(t *testing.T) {
		cfg := &ClientConfig{}
		if cfg.AccessToken != "" {
			t.Errorf("Expected AccessToken to be empty string, got '%s'", cfg.AccessToken)
		}
	})

	t.Run("ClientConfig can be initialized with access token", func(t *testing.T) {
		expectedToken := "test-access-token"
		cfg := &ClientConfig{
			AccessToken: expectedToken,
		}
		if cfg.AccessToken != expectedToken {
			t.Errorf("Expected AccessToken to be '%s', got '%s'", expectedToken, cfg.AccessToken)
		}
	})
}

func TestAbsPath(t *testing.T) {
	t.Run("AbsPath returns valid path", func(t *testing.T) {
		path, err := AbsPath()
		if err != nil {
			t.Fatalf("AbsPath returned error: %v", err)
		}
		if path == "" {
			t.Fatal("AbsPath returned empty path")
		}

		// Verify the path ends with the expected filename
		expectedFilename := ClientConfigFileName
		if filepath.Base(path) != expectedFilename {
			t.Errorf("Expected path to end with '%s', got '%s'", expectedFilename, filepath.Base(path))
		}
	})

	t.Run("AbsPath returns path in home directory", func(t *testing.T) {
		path, err := AbsPath()
		if err != nil {
			t.Fatalf("AbsPath returned error: %v", err)
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Failed to get home directory: %v", err)
		}

		expectedPath := filepath.Join(homeDir, ClientConfigFileName)
		if path != expectedPath {
			t.Errorf("Expected path to be '%s', got '%s'", expectedPath, path)
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("Save creates file with valid config", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			os.Setenv("HOME", originalHome)
		}()

		// Set HOME to temp directory
		os.Setenv("HOME", tempDir)

		cfg := &ClientConfig{
			AccessToken: "test-token-123",
		}

		err := Save(cfg)
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}

		// Verify file was created
		path, err := AbsPath()
		if err != nil {
			t.Fatalf("AbsPath returned error: %v", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatal("Config file was not created")
		}

		// Verify file contents
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}

		// Check that the content contains the access token
		contentStr := string(content)
		if !testhelpers.Contains(contentStr, "test-token-123") {
			t.Errorf("Config file does not contain expected access token. Content: %s", contentStr)
		}
	})

	t.Run("Save handles nil config gracefully", func(t *testing.T) {
		// This test would require checking if Save handles nil config
		// The current implementation doesn't explicitly handle nil, so it might panic
		// This is a placeholder for future error handling improvements
	})

	t.Run("Save overwrites existing file", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			os.Setenv("HOME", originalHome)
		}()

		// Set HOME to temp directory
		os.Setenv("HOME", tempDir)

		// Create initial config
		cfg1 := &ClientConfig{
			AccessToken: "first-token",
		}
		err := Save(cfg1)
		if err != nil {
			t.Fatalf("First Save returned error: %v", err)
		}

		// Overwrite with new config
		cfg2 := &ClientConfig{
			AccessToken: "second-token",
		}
		err = Save(cfg2)
		if err != nil {
			t.Fatalf("Second Save returned error: %v", err)
		}

		// Verify file contains new content
		path, err := AbsPath()
		if err != nil {
			t.Fatalf("AbsPath returned error: %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}

		contentStr := string(content)
		if !testhelpers.Contains(contentStr, "second-token") {
			t.Errorf("Config file does not contain expected second token. Content: %s", contentStr)
		}
		if testhelpers.Contains(contentStr, "first-token") {
			t.Error("Config file still contains old token")
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("Load returns empty config when file does not exist", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			os.Setenv("HOME", originalHome)
		}()

		// Set HOME to temp directory (no config file exists)
		os.Setenv("HOME", tempDir)

		cfg := Load()
		if cfg == nil {
			t.Fatal("Load returned nil config")
		}
		if cfg.AccessToken != "" {
			t.Errorf("Expected empty AccessToken, got '%s'", cfg.AccessToken)
		}
	})

	t.Run("Load returns config from existing file", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			os.Setenv("HOME", originalHome)
		}()

		// Set HOME to temp directory
		os.Setenv("HOME", tempDir)

		// Create and save a config
		expectedToken := "loaded-token-456"
		cfgToSave := &ClientConfig{
			AccessToken: expectedToken,
		}
		err := Save(cfgToSave)
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}

		// Load the config
		loadedCfg := Load()
		if loadedCfg == nil {
			t.Fatal("Load returned nil config")
		}
		if loadedCfg.AccessToken != expectedToken {
			t.Errorf("Expected AccessToken to be '%s', got '%s'", expectedToken, loadedCfg.AccessToken)
		}
	})

	t.Run("Load handles invalid YAML gracefully", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			os.Setenv("HOME", originalHome)
		}()

		// Set HOME to temp directory
		os.Setenv("HOME", tempDir)

		// Create invalid YAML file
		path, err := AbsPath()
		if err != nil {
			t.Fatalf("AbsPath returned error: %v", err)
		}

		invalidYAML := "invalid: yaml: content: ["
		err = os.WriteFile(path, []byte(invalidYAML), 0o644)
		if err != nil {
			t.Fatalf("Failed to write invalid YAML: %v", err)
		}

		// Load should not panic and return empty config
		cfg := Load()
		if cfg == nil {
			t.Fatal("Load returned nil config")
		}
		if cfg.AccessToken != "" {
			t.Errorf("Expected empty AccessToken for invalid YAML, got '%s'", cfg.AccessToken)
		}
	})

	t.Run("Load handles empty file gracefully", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			os.Setenv("HOME", originalHome)
		}()

		// Set HOME to temp directory
		os.Setenv("HOME", tempDir)

		// Create empty file
		path, err := AbsPath()
		if err != nil {
			t.Fatalf("AbsPath returned error: %v", err)
		}

		err = os.WriteFile(path, []byte(""), 0o644)
		if err != nil {
			t.Fatalf("Failed to write empty file: %v", err)
		}

		// Load should not panic and return empty config
		cfg := Load()
		if cfg == nil {
			t.Fatal("Load returned nil config")
		}
		if cfg.AccessToken != "" {
			t.Errorf("Expected empty AccessToken for empty file, got '%s'", cfg.AccessToken)
		}
	})
}

func TestConfigIntegration(t *testing.T) {
	t.Run("Save and Load round trip", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			os.Setenv("HOME", originalHome)
		}()

		// Set HOME to temp directory
		os.Setenv("HOME", tempDir)

		// Save a config
		originalCfg := &ClientConfig{
			AccessToken: "round-trip-token-789",
		}
		err := Save(originalCfg)
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}

		// Load the config
		loadedCfg := Load()
		if loadedCfg == nil {
			t.Fatal("Load returned nil config")
		}

		// Verify they match
		if loadedCfg.AccessToken != originalCfg.AccessToken {
			t.Errorf("Round trip failed. Original: '%s', Loaded: '%s'",
				originalCfg.AccessToken, loadedCfg.AccessToken)
		}
	})
}
