// Package config provides configuration management functionality for the MCPJungle application.
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ClientConfigFileName = ".mcpjungle.conf"

// ClientConfig represents the MCPJungle client configuration stored in the user's home directory.
// It can contain configuration for both a standard user and an admin user.
type ClientConfig struct {
	AccessToken string `yaml:"access_token"`
}

// AbsPath returns the absolute path to the client configuration file.
// It combines the user's home directory with the ClientConfigFileName.
// The path is returned regardless of whether the file actually exists there or not.
func AbsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ClientConfigFileName), nil
}

// Save saves the ClientConfig to the file system at AbsPath().
// If the file does not exist, this method creates it.
func Save(c *ClientConfig) error {
	path, err := AbsPath()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	defer encoder.Close()
	return encoder.Encode(c)
}

// Load loads the client configuration from the user's home directory on best-effort basis.
// If this function encounters any errors (or the config does not exist), it simply returns an empty ClientConfig.
func Load() *ClientConfig {
	cfg := &ClientConfig{}

	path, err := AbsPath()
	if err != nil {
		return cfg
	}

	f, err := os.Open(path)
	if err != nil {
		return cfg
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	_ = decoder.Decode(cfg)

	return cfg
}

// ReadFile reads configuration from an external file and unmarshals it into the provided struct pointer.
// The struct should have appropriate json and yaml tags.
// Supports both JSON (.json) and YAML (.yaml, .yml) file formats.
func ReadFile[T any](filePath string, config *T) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	ext := filepath.Ext(filePath)
	switch ext {
	case ".yaml", ".yml":
		return readFileYaml(f, config)
	case ".json":
		return readFileJson(f, config)
	default:
		fmt.Println("Unknown config file extension. Assuming json format.")
		return readFileJson(f, config)
	}
}

// readFileJson reads and unmarshals JSON configuration from a reader.
func readFileJson[T any](reader io.Reader, config *T) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config from config file: %w", err)
	}

	return nil
}

// readFileYaml reads and unmarshals YAML configuration from a reader.
func readFileYaml[T any](reader io.Reader, config *T) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read config from config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config from config file: %w", err)
	}

	return nil
}
