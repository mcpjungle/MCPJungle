package mcp

import (
	"context"
	"fmt"
	"log"

	"github.com/mcpjungle/mcpjungle/internal/model"
)

// RegisterMcpServer registers a new MCP server in the database.
// It also registers all the Tools and Prompts provided by the server.
// Tool and prompt registration is on best-effort basis and does not fail the server registration.
// Registered tools and prompts are also added to the MCP proxy server.
func (m *MCPService) RegisterMcpServer(ctx context.Context, s *model.McpServer) error {
	if err := validateServerName(s.Name); err != nil {
		return err
	}

	mcpClient, err := newMcpServerSession(ctx, s)
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
	if err = m.registerServerPrompts(ctx, s, mcpClient); err != nil {
		log.Printf("[WARN] failed to register prompts for MCP server %s: %v", s.Name, err)
	}

	return nil
}

// DeregisterMcpServer deregisters an MCP server from the database.
// It also deregisters all the tools and prompts registered by the server.
// If even a single tool or prompt fails to deregister, the server deregistration fails.
// Deregistered tools and prompts are also removed from the MCP proxy server.
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
	if err := m.db.Unscoped().Delete(s).Error; err != nil {
		return fmt.Errorf("failed to deregister server %s: %w", name, err)
	}

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
		return nil, err
	}
	return &serverModel, nil
}
