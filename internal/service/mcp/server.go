package mcp

import (
	"context"
	"errors"
	"fmt"
	"log"

	mcpgotransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

// RegisterMcpServerWithOAuthSupport attempts to register a server immediately and,
// if upstream OAuth authorization is required, persists a pending auth session that
// can be completed later.
func (m *MCPService) RegisterMcpServerWithOAuthSupport(
	ctx context.Context,
	input *types.RegisterServerInput,
	s *model.McpServer,
	force bool,
	initiatedBy string,
) error {
	// First attempt to register the server without involving any oauth flows.
	// This covers all mcp servers that DO NOT specifically require oauth-based authentication.
	err := m.registerMcpServerWithoutOAuth(ctx, s)
	if err == nil {
		return nil
	}

	// If registration failed and the error is not related to oauth, return error.
	if s.Transport != types.TransportStreamableHTTP && s.Transport != types.TransportSSE {
		return err
	}
	if !errors.Is(err, mcpgotransport.ErrUnauthorized) {
		return err
	}

	// registration failed due to missing/invalid upstream OAuth credentials.
	// notify the client so the oauth flow can be initiated.
	return m.bootstrapUpstreamOAuth(ctx, input, s, force, initiatedBy)
}

// registerMcpServerWithoutOAuth performs the initial upstream registration
// attempt without attaching any stored upstream OAuth credentials.
func (m *MCPService) registerMcpServerWithoutOAuth(ctx context.Context, s *model.McpServer) error {
	return m.registerMcpServer(ctx, s, false)
}

// finalizeMcpServerRegistration performs the plain MCP server registration flow
// once upstream authentication has already been satisfied and any stored
// upstream OAuth credentials should be attached to the connection attempt.
func (m *MCPService) finalizeMcpServerRegistration(ctx context.Context, s *model.McpServer) error {
	return m.registerMcpServer(ctx, s, true)
}

// registerMcpServer performs the core MCP server registration flow.
//
// It first registers the MCP server in the DB, then registers all the Tools,
// Prompts, and Resources provided by the server. Tool registration is required,
// while prompt/resource registration is best-effort. Registered entities are
// also added to the MCP proxy server.
//
// This method assumes that any Oauth nuance is already handled and simply uses existing auth info.
func (m *MCPService) registerMcpServer(ctx context.Context, s *model.McpServer, useStoredUpstreamAuth bool) error {
	if err := validateServerName(s.Name); err != nil {
		return err
	}

	// Only validate URLs for transports that actually carry a URL in their config.
	switch s.Transport {
	case types.TransportStreamableHTTP:
		conf, err := s.GetStreamableHTTPConfig()
		if err != nil {
			return err
		}
		if err := validateURL(conf.URL); err != nil {
			return err
		}
	case types.TransportSSE:
		conf, err := s.GetSSEConfig()
		if err != nil {
			return err
		}
		if err := validateURL(conf.URL); err != nil {
			return err
		}
	}

	mcpClient, err := createMcpServerConnectionWithDB(
		ctx,
		m.db,
		s,
		m.mcpServerInitReqTimeoutSec,
		useStoredUpstreamAuth,
	)
	if err != nil {
		return err
	}
	defer mcpClient.Close()

	// register the server in the DB
	if err := m.db.Create(s).Error; err != nil {
		return fmt.Errorf("failed to register mcp server: %w", err)
	}

	if err = m.registerServerTools(ctx, s, mcpClient); err != nil {
		return fmt.Errorf("failed to register tools for MCP server %s: %w", s.Name, err)
	}

	// Register prompts (best-effort, don't fail server registration)
	if mcpClient.GetServerCapabilities().Prompts != nil {
		if err = m.registerServerPrompts(ctx, s, mcpClient); err != nil {
			log.Printf("[WARN] failed to register prompts for MCP server %s: %v", s.Name, err)
		}
	}
	if mcpClient.GetServerCapabilities().Resources != nil {
		if err = m.registerServerResources(ctx, s, mcpClient); err != nil {
			log.Printf("[WARN] failed to register resources for MCP server %s: %v", s.Name, err)
		}
	}

	return nil
}

// DeregisterMcpServer deregisters an MCP server from the database.
// It also deregisters all the tools, prompts and resources registered by the server.
// If even a single tool, prompt or resource fails to deregister, the server deregistration fails.
// Deregistered tools, prompts and resources are also removed from the MCP proxy server.
// Any stateful sessions associated with this server are also closed.
func (m *MCPService) DeregisterMcpServer(name string) error {
	s, err := m.GetMcpServer(name)
	if err != nil {
		return fmt.Errorf("failed to get MCP server %s from DB: %w", name, err)
	}
	if err := m.deregisterServerTools(s); err != nil {
		return fmt.Errorf(
			"failed to deregister tools for server %s, cannot proceed with server deregistration: %w",
			name,
			err,
		)
	}
	if err := m.deregisterServerPrompts(s); err != nil {
		return fmt.Errorf(
			"failed to deregister prompts for server %s, cannot proceed with server deregistration: %w",
			name,
			err,
		)
	}
	if err := m.deregisterServerResources(s); err != nil {
		return fmt.Errorf(
			"failed to deregister resources for server %s, cannot proceed with server deregistration: %w",
			name,
			err,
		)
	}
	if err := m.db.Unscoped().Delete(s).Error; err != nil {
		return fmt.Errorf("failed to deregister server %s: %w", name, err)
	}
	if err := m.db.Unscoped().Where("server_name = ?", name).Delete(&model.UpstreamOAuthToken{}).Error; err != nil {
		return fmt.Errorf("failed to remove upstream OAuth tokens for server %s: %w", name, err)
	}
	if err := m.db.Unscoped().Where("server_name = ?", name).Delete(&model.UpstreamOAuthPendingSession{}).Error; err != nil {
		return fmt.Errorf("failed to remove pending upstream OAuth sessions for server %s: %w", name, err)
	}

	// Close any stateful session associated with this server
	m.sessionManager.CloseSession(name)

	return nil
}

// ListMcpServers returns all registered MCP servers.
func (m *MCPService) ListMcpServers() ([]model.McpServer, error) {
	var servers []model.McpServer
	if err := m.db.Find(&servers).Error; err != nil {
		return nil, err
	}
	return servers, nil
}

// GetMcpServer fetches a server from the database by name.
func (m *MCPService) GetMcpServer(name string) (*model.McpServer, error) {
	var serverModel model.McpServer
	if err := m.db.Where("name = ?", name).First(&serverModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("MCP server %s not found: %w", name, apierrors.ErrNotFound)
		}
		return nil, err
	}
	return &serverModel, nil
}

// EnableMcpServer enables all tools, prompts and resources registered by the given MCP server.
// It returns the names of the enabled tools and prompts.
// If even a single tool, prompt or resource fails to enable, the operation fails.
func (m *MCPService) EnableMcpServer(name string) ([]string, []string, error) {
	if err := validateServerName(name); err != nil {
		return nil, nil, err
	}
	toolsEnabled, err := m.EnableTools(name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to enable tools for server %s: %w", name, err)
	}
	promptsEnabled, err := m.EnablePrompts(name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to enable prompts for server %s: %w", name, err)
	}
	if _, err := m.EnableResources(name); err != nil {
		return nil, nil, fmt.Errorf("failed to enable resources for server %s: %w", name, err)
	}
	return toolsEnabled, promptsEnabled, nil
}

// DisableMcpServer disables all tools, prompts and resources registered by the given MCP server.
// It returns the names of the disabled tools and prompts.
// If even a single tool, prompt or resource fails to disable, the operation fails.
func (m *MCPService) DisableMcpServer(name string) ([]string, []string, error) {
	if err := validateServerName(name); err != nil {
		return nil, nil, err
	}
	toolsDisabled, err := m.DisableTools(name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to disable tools for server %s: %w", name, err)
	}
	promptsDisabled, err := m.DisablePrompts(name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to disable prompts for server %s: %w", name, err)
	}
	if _, err := m.DisableResources(name); err != nil {
		return nil, nil, fmt.Errorf("failed to disable resources for server %s: %w", name, err)
	}
	return toolsDisabled, promptsDisabled, nil
}
