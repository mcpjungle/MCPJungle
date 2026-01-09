// Package mcp provides MCP (Model Context Protocol) service functionality for the MCPJungle application.
package mcp

import (
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"gorm.io/gorm"
)

// ServiceConfig holds the configuration parameters for initializing the MCPService.
type ServiceConfig struct {
	DB *gorm.DB

	McpProxyServer    *server.MCPServer
	SseMcpProxyServer *server.MCPServer

	Metrics telemetry.CustomMetrics

	McpServerInitReqTimeout int

	// SessionManager manages persistent connections for MCP servers configured in stateful mode.
	// If nil, a default SessionManager will be created.
	SessionManager *SessionManager
}

// MCPService coordinates operations amongst the registry database, mcp proxy server and upstream MCP servers.
// It is responsible for maintaining data consistency and providing a unified interface for MCP operations.
type MCPService struct {
	db *gorm.DB

	mcpProxyServer    *server.MCPServer
	sseMcpProxyServer *server.MCPServer

	// toolInstances keeps track of all the in-memory mcp.Tool instances, keyed by their unique names.
	toolInstances map[string]mcp.Tool
	mu            sync.RWMutex

	// toolDeletionCallback is a callback that gets invoked when one or more tools is removed
	// (deregistered or disabled) from mcpjungle.
	toolDeletionCallback ToolDeletionCallback
	// toolAdditionCallback is a callback that gets invoked when one or more tools is added
	// (registered or (re)enabled) in mcpjungle.
	toolAdditionCallback ToolAdditionCallback

	metrics telemetry.CustomMetrics

	mcpServerInitReqTimeoutSec int

	// sessionManager manages persistent connections for MCP servers configured in stateful mode.
	sessionManager *SessionManager
}

// NewMCPService creates a new instance of MCPService.
// It initializes the MCP proxy server by loading all registered tools from the database.
func NewMCPService(c *ServiceConfig) (*MCPService, error) {
	// Use the provided session manager, or create a default one if not provided
	sessionManager := c.SessionManager
	if sessionManager == nil {
		sessionManager = NewSessionManager(&SessionManagerConfig{
			IdleTimeoutSec:    DefaultSessionIdleTimeoutSec,
			InitReqTimeoutSec: c.McpServerInitReqTimeout,
		})
	}

	s := &MCPService{
		db: c.DB,

		mcpProxyServer:    c.McpProxyServer,
		sseMcpProxyServer: c.SseMcpProxyServer,

		toolInstances: make(map[string]mcp.Tool),
		mu:            sync.RWMutex{},

		// initialize the callbacks to NOOP functions
		toolDeletionCallback: func(toolNames ...string) {},
		toolAdditionCallback: func(toolName string) error { return nil },

		metrics: c.Metrics,

		mcpServerInitReqTimeoutSec: c.McpServerInitReqTimeout,

		sessionManager: sessionManager,
	}
	if err := s.initMCPProxyServer(); err != nil {
		return nil, fmt.Errorf("failed to initialize MCP proxy server: %w", err)
	}
	return s, nil
}

// Shutdown gracefully shuts down the MCP service, closing all stateful sessions.
func (m *MCPService) Shutdown() {
	if m.sessionManager != nil {
		m.sessionManager.Shutdown()
	}
}
