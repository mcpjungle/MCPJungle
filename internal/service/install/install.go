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
	httpClient *http.Client
}

// NewInstallService creates a new InstallService instance.
func NewInstallService(mcpService *mcp.MCPService, httpClient *http.Client) *InstallService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &InstallService{
		mcpService: mcpService,
		httpClient: httpClient,
	}
}

// InstallFromRegistry installs an MCP server from the official registry.
func (s *InstallService) InstallFromRegistry(ctx context.Context, options *types.InstallOptions) (*model.McpServer, error) {
	// Get server registry information
	registry, err := s.getServerRegistry(options.ServerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get server registry info: %w", err)
	}

	// Convert registry info to RegisterServerInput
	registerInput, err := s.convertRegistryToRegisterInput(registry, options)
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

// getServerRegistry fetches server information from the built-in registry.
func (s *InstallService) getServerRegistry(serverName string) (*types.ServerRegistry, error) {
	// Built-in registry of official MCP servers
	registry := s.getBuiltinRegistry()

	server, exists := registry[serverName]
	if !exists {
		return nil, fmt.Errorf("server '%s' not found in registry. Available servers: %s",
			serverName, strings.Join(s.getAvailableServers(), ", "))
	}

	return &server, nil
}

// getBuiltinRegistry returns the built-in registry of official MCP servers.
func (s *InstallService) getBuiltinRegistry() map[string]types.ServerRegistry {
	return map[string]types.ServerRegistry{
		"time": {
			Name:           "time",
			Package:        "@modelcontextprotocol/server-time",
			Transport:      string(types.TransportStdio),
			Command:        "npx",
			Args:           []string{"-y", "@modelcontextprotocol/server-time"},
			Description:    "Get the current time and date",
			Category:       "utility",
			PackageManager: "npm",
		},
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
			Args:           []string{"-y", "@modelcontextprotocol/server-filesystem"},
			Description:    "Read and write files on the local filesystem",
			Category:       "filesystem",
			PackageManager: "npm",
		},
		"git": {
			Name:           "git",
			Package:        "mcp-server-git",
			Transport:      string(types.TransportStdio),
			Command:        "uvx",
			Args:           []string{"mcp-server-git"},
			Description:    "Git repository operations",
			Category:       "development",
			PackageManager: "pip",
		},
		"github": {
			Name:           "github",
			Package:        "@modelcontextprotocol/server-github",
			Transport:      string(types.TransportStdio),
			Command:        "npx",
			Args:           []string{"-y", "@modelcontextprotocol/server-github"},
			Description:    "GitHub repository and issue management",
			Category:       "development",
			PackageManager: "npm",
		},
		"postgres": {
			Name:           "postgres",
			Package:        "@modelcontextprotocol/server-postgres",
			Transport:      string(types.TransportStdio),
			Command:        "npx",
			Args:           []string{"-y", "@modelcontextprotocol/server-postgres"},
			Description:    "PostgreSQL database operations",
			Category:       "database",
			PackageManager: "npm",
		},
		"sqlite": {
			Name:           "sqlite",
			Package:        "@modelcontextprotocol/server-sqlite",
			Transport:      string(types.TransportStdio),
			Command:        "npx",
			Args:           []string{"-y", "@modelcontextprotocol/server-sqlite"},
			Description:    "SQLite database operations",
			Category:       "database",
			PackageManager: "npm",
		},
		"brave-search": {
			Name:           "brave-search",
			Package:        "@modelcontextprotocol/server-brave-search",
			Transport:      string(types.TransportStdio),
			Command:        "npx",
			Args:           []string{"-y", "@modelcontextprotocol/server-brave-search"},
			Description:    "Search the web using Brave Search API",
			Category:       "web",
			PackageManager: "npm",
		},
		"fetch": {
			Name:           "fetch",
			Package:        "@modelcontextprotocol/server-fetch",
			Transport:      string(types.TransportStdio),
			Command:        "npx",
			Args:           []string{"-y", "@modelcontextprotocol/server-fetch"},
			Description:    "Fetch web pages and extract content",
			Category:       "web",
			PackageManager: "npm",
		},
		"puppeteer": {
			Name:           "puppeteer",
			Package:        "@modelcontextprotocol/server-puppeteer",
			Transport:      string(types.TransportStdio),
			Command:        "npx",
			Args:           []string{"-y", "@modelcontextprotocol/server-puppeteer"},
			Description:    "Web automation and scraping with Puppeteer",
			Category:       "web",
			PackageManager: "npm",
		},
	}
}

// getAvailableServers returns a list of available server names.
func (s *InstallService) getAvailableServers() []string {
	registry := s.getBuiltinRegistry()
	servers := make([]string, 0, len(registry))
	for name := range registry {
		servers = append(servers, name)
	}
	return servers
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

	// Handle version if specified
	if options.Version != "" {
		// For npm packages, add version to the package name
		if registry.PackageManager == "npm" {
			packageWithVersion := fmt.Sprintf("%s@%s", registry.Package, options.Version)
			// Replace the package name in args
			for i, arg := range args {
				if strings.Contains(arg, registry.Package) {
					args[i] = strings.Replace(arg, registry.Package, packageWithVersion, 1)
					break
				}
			}
		}
		// For pip packages, add version to the package name
		if registry.PackageManager == "pip" {
			packageWithVersion := fmt.Sprintf("%s==%s", registry.Package, options.Version)
			// Replace the package name in args
			for i, arg := range args {
				if strings.Contains(arg, registry.Package) {
					args[i] = strings.Replace(arg, registry.Package, packageWithVersion, 1)
					break
				}
			}
		}
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
