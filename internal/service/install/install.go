// Package install provides functionality for installing MCP servers from the official registry.
package install

import (
	"context"
	"fmt"
	"net/http"
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

// InstallFromRegistry installs an MCP server from the official registry.
func (s *InstallService) InstallFromRegistry(ctx context.Context, options *types.InstallOptions) (*model.McpServer, error) {
	// Simple hardcoded registry
	registry := map[string]types.ServerRegistry{
		"memory": {
			Name:           "memory",
			Package:        "@modelcontextprotocol/server-memory",
			Transport:      string(types.TransportStdio),
			Command:        "npx",
			Args:           []string{"-y", "@modelcontextprotocol/server-memory"},
			Description:    "Persistent memory for conversations",
			Category:       "utility",
			PackageManager: "npm",
		},
		"filesystem": {
			Name:           "filesystem",
			Package:        "@modelcontextprotocol/server-filesystem",
			Transport:      string(types.TransportStdio),
			Command:        "npx",
			Args:           []string{"-y", "@modelcontextprotocol/server-filesystem", "."},
			Description:    "Read and write files on the local filesystem",
			Category:       "filesystem",
			PackageManager: "npm",
		},
		"sequentialthinking": {
			Name:           "sequentialthinking",
			Package:        "@modelcontextprotocol/server-sequential-thinking",
			Transport:      string(types.TransportStdio),
			Command:        "npx",
			Args:           []string{"-y", "@modelcontextprotocol/server-sequential-thinking"},
			Description:    "MCP server for sequential thinking and problem solving",
			Category:       "utility",
			PackageManager: "npm",
		},
		"everything": {
			Name:           "everything",
			Package:        "@modelcontextprotocol/server-everything",
			Transport:      string(types.TransportStdio),
			Command:        "npx",
			Args:           []string{"-y", "@modelcontextprotocol/server-everything"},
			Description:    "MCP server that provides access to everything",
			Category:       "utility",
			PackageManager: "npm",
		},
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
	}, nil
}
