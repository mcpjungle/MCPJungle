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

	// toolDeletionCallbacks holds a list of callbacks to be invoked when one or more tools is
	// deregistered from mcpjungle.
	toolDeletionCallbacks []ToolDeletionCallback
	callbackMu            sync.RWMutex
}

// NewMCPService creates a new instance of MCPService.
// It initializes the MCP proxy server by loading all registered tools from the database.
func NewMCPService(db *gorm.DB, mcpProxyServer *server.MCPServer) (*MCPService, error) {
	s := &MCPService{
		db:             db,
		mcpProxyServer: mcpProxyServer,
		toolInstances:  make(map[string]mcp.Tool),
		mu:             sync.RWMutex{},
	}
	if err := s.initMCPProxyServer(); err != nil {
		return nil, fmt.Errorf("failed to initialize MCP proxy server: %w", err)
	}
	return s, nil
}
