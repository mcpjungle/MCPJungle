package mcp

import (
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gorm.io/gorm"
	"sync"
)

// MCPService coordinates operations amongst the registry database, mcp proxy server and upstream MCP servers.
// It is responsible for maintaining data consistency and providing a unified interface for MCP operations.
type MCPService struct {
	db             *gorm.DB
	mcpProxyServer *server.MCPServer

	// toolInstances keeps track of all the in-memory mcp.Tool instances, keyed by their unique names.
	toolInstances map[string]mcp.Tool
	mu            sync.RWMutex

	// toolDeletionCallback is a callback that gets invoked when one or more tools is removed
	// (deregistered or disabled) from mcpjungle.
	toolDeletionCallback ToolDeletionCallback
	// toolAdditionCallback is a callback that gets invoked when one or more tools is added
	// (registered or (re)enabled) in mcpjungle.
	toolAdditionCallback ToolAdditionCallback
}

// NewMCPService creates a new instance of MCPService.
// It initializes the MCP proxy server by loading all registered tools from the database.
func NewMCPService(db *gorm.DB, mcpProxyServer *server.MCPServer) (*MCPService, error) {
	s := &MCPService{
		db:             db,
		mcpProxyServer: mcpProxyServer,

		toolInstances: make(map[string]mcp.Tool),
		mu:            sync.RWMutex{},

		// initialize the callbacks to NOOP functions
		toolDeletionCallback: func(toolNames ...string) {},
		toolAdditionCallback: func(toolName string) error { return nil },
	}
	if err := s.initMCPProxyServer(); err != nil {
		return nil, fmt.Errorf("failed to initialize MCP proxy server: %w", err)
	}
	return s, nil
}
