// Package install provides functionality for installing MCP servers from the official registry.
package install

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// InstallService handles installation of MCP servers from the official registry.
type InstallService struct {
	mcpService *mcp.MCPService
}

// NewInstallService creates a new InstallService instance.
func NewInstallService(mcpService *mcp.MCPService, httpClient *http.Client) *InstallService {
	return &InstallService{
		mcpService: mcpService,
	}
}

// ServerConfig represents the structure of the servers.json file
type ServerConfig struct {
	Servers map[string]types.ServerRegistry `json:"servers"`
}

// InstallFromRegistry installs an MCP server from the official registry.
func (s *InstallService) InstallFromRegistry(ctx context.Context, options *types.InstallOptions) (*model.McpServer, error) {
	// Load server registry from JSON file
	registry, err := s.LoadServerRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load server registry: %w", err)
	}

	// Find the requested server
	serverConfig, exists := registry[options.ServerName]
	if !exists {
		// Get available server names
		availableServers := make([]string, 0, len(registry))
		for name := range registry {
			availableServers = append(availableServers, name)
		}
		return nil, fmt.Errorf("server '%s' not found in registry. Available servers: %s",
			options.ServerName, strings.Join(availableServers, ", "))
	}

	// Check required environment variables
	for _, envVar := range serverConfig.RequiredEnvVars {
		if _, exists := options.Env[envVar]; !exists {
			return nil, fmt.Errorf("required environment variable %s not provided. Use --env %s=value", envVar, envVar)
		}
	}

	// Validate dependencies for local setup only
	if err := s.validateDependencies(&serverConfig); err != nil {
		return nil, fmt.Errorf("dependency validation failed: %w", err)
	}

	// Add optional environment variables with defaults
	env := make(map[string]string)
	for k, v := range options.Env {
		env[k] = v
	}
	for envVar, defaultValue := range serverConfig.OptionalEnvVars {
		if _, exists := env[envVar]; !exists {
			env[envVar] = defaultValue
		}
	}

	// Convert registry info to RegisterServerInput
	registerInput, err := s.convertRegistryToRegisterInput(&serverConfig, options)
	if err != nil {
		return nil, fmt.Errorf("failed to convert registry info: %w", err)
	}

	// Create the server model
	var server *model.McpServer
	switch types.McpServerTransport(registerInput.Transport) {
	case types.TransportStdio:
		server, err = model.NewStdioServer(
			registerInput.Name,
			registerInput.Description,
			registerInput.Command,
			registerInput.Args,
			registerInput.Env,
		)
	case types.TransportStreamableHTTP:
		server, err = model.NewStreamableHTTPServer(
			registerInput.Name,
			registerInput.Description,
			registerInput.URL,
			registerInput.BearerToken,
		)
	case types.TransportSSE:
		server, err = model.NewSSEServer(
			registerInput.Name,
			registerInput.Description,
			registerInput.URL,
			registerInput.BearerToken,
		)
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", registerInput.Transport)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create server model: %w", err)
	}

	// Register the server using the existing MCP service
	if err := s.mcpService.RegisterMcpServer(ctx, server); err != nil {
		return nil, fmt.Errorf("failed to register server: %w", err)
	}

	return server, nil
}

// convertRegistryToRegisterInput converts registry information to RegisterServerInput.
func (s *InstallService) convertRegistryToRegisterInput(registry *types.ServerRegistry, options *types.InstallOptions) (*types.RegisterServerInput, error) {
	// Start with registry args and merge with user-provided args
	args := make([]string, len(registry.Args))
	copy(args, registry.Args)
	args = append(args, options.Args...)

	// Start with registry env and merge with user-provided env
	env := make(map[string]string)
	for k, v := range options.Env {
		env[k] = v
	}

	return &types.RegisterServerInput{
		Name:        registry.Name,
		Transport:   registry.Transport,
		Description: registry.Description,
		Command:     registry.Command,
		Args:        args,
		Env:         env,
		URL:         registry.URL,
		BearerToken: registry.BearerToken,
	}, nil
}

// validateDependencies checks if required dependencies are available.
func (s *InstallService) validateDependencies(serverConfig *types.ServerRegistry) error {
	// Validate based on package manager
	switch serverConfig.PackageManager {
	case "npm":
		if !s.isCommandAvailable("npx") {
			return fmt.Errorf("npx is required but not installed. Please install Node.js from https://nodejs.org/ (includes npm and npx)")
		}
	case "pip":
		if !s.isCommandAvailable("python") {
			return fmt.Errorf("python is required but not installed. Please install Python from https://python.org/")
		}
		if !s.isCommandAvailable("pip") {
			return fmt.Errorf("pip is required but not installed. Please install pip (usually comes with Python)")
		}
	case "none":
		// No dependencies needed for HTTP servers
		return nil
	default:
		return fmt.Errorf("unsupported package manager: %s", serverConfig.PackageManager)
	}

	return nil
}

// isCommandAvailable checks if a command is available in the system PATH.
func (s *InstallService) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// LoadServerRegistry loads the server registry from the JSON configuration file.
func (s *InstallService) LoadServerRegistry() (map[string]types.ServerRegistry, error) {
	// Read the config file from the project directory or container path
	configPaths := []string{
		"mcp-config/servers.json",      // Local development
		"/app/mcp-config/servers.json", // Docker container
	}

	var configData []byte
	var err error
	for _, path := range configPaths {
		configData, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("could not read mcp-config/servers.json: %w", err)
	}

	var config ServerConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse servers.json: %w", err)
	}

	return config.Servers, nil
}
