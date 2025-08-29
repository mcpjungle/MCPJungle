package toolgroup

import (
	"errors"
	"fmt"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"gorm.io/gorm"
	"sync"
)

// ToolGroupService provides methods to manage tool groups and their associated MCP proxy servers.
type ToolGroupService struct {
	db *gorm.DB

	mcpService *mcp.MCPService

	// mcpServers manages the MCP proxy servers for all the tool groups
	mcpServers map[string]*server.MCPServer
	// mu protects access to the mcpServers map
	mu sync.RWMutex
}

func NewToolGroupService(db *gorm.DB, mcpService *mcp.MCPService) (*ToolGroupService, error) {
	s := &ToolGroupService{
		db:         db,
		mcpService: mcpService,
		mcpServers: make(map[string]*server.MCPServer),
		mu:         sync.RWMutex{},
	}

	// register callback to handle when a tool gets deleted
	mcpService.RegisterToolDeletionCallback(s.handleToolDeletion)

	if err := s.initToolGroupMCPServers(); err != nil {
		return nil, fmt.Errorf("failed to initialize tool group MCP servers: %w", err)
	}
	return s, nil
}

// CreateToolGroup creates a new tool group in the database and a Proxy MCP server that just exposes the specified tools.
func (s *ToolGroupService) CreateToolGroup(group *model.ToolGroup) error {
	toolNames, err := group.GetTools()
	if err != nil {
		return fmt.Errorf("failed to parse toolNames: %w", err)
	}
	if len(toolNames) == 0 {
		return errors.New("tool group must contain at least one tool")
	}

	// create the proxy MCP server that exposes only specified tools
	mcpServer := s.newMCPServer(group.Name)
	// populate the MCP server with the specified tools
	for _, name := range toolNames {
		tool, exists := s.mcpService.GetToolInstance(name)
		if !exists {
			return fmt.Errorf("tool %s does not exist in the tool instances tracker", name)
		}
		mcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
	}

	// first, add the tool group to the database
	// this also checks for uniqueness of the group's name
	if err := s.db.Create(group).Error; err != nil {
		return fmt.Errorf("failed to create tool group: %w", err)
	}

	// finally, add the proxy MCP to the tool group MCPs manager so that it is ready to serve
	s.addToolGroupMCPServer(group.Name, mcpServer)

	return nil
}

// GetToolGroup retrieves a tool group by name from the database.
func (s *ToolGroupService) GetToolGroup(name string) (*model.ToolGroup, error) {
	var group model.ToolGroup
	if err := s.db.Where("name = ?", name).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

// ListToolGroups retrieves all tool groups from the database.
func (s *ToolGroupService) ListToolGroups() ([]model.ToolGroup, error) {
	var groups []model.ToolGroup
	if err := s.db.Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *ToolGroupService) DeleteToolGroup(name string) error {
	s.deleteToolGroupMCPServer(name)

	err := s.db.Unscoped().Where("name = ?", name).Delete(&model.ToolGroup{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete toolgroup: %w", err)
	}
	return nil
}

// GetToolGroupMCPServer retrieves the MCP proxy server for a given tool group name.
// This method is safe to call concurrently.
func (s *ToolGroupService) GetToolGroupMCPServer(name string) (*server.MCPServer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mcpServer, exists := s.mcpServers[name]
	return mcpServer, exists
}

// newMCPServer creates a new MCP proxy server for a given tool group name.
func (s *ToolGroupService) newMCPServer(groupName string) *server.MCPServer {
	return server.NewMCPServer(
		fmt.Sprintf("MCPJungle proxy MCP server for tool group: %s", groupName),
		"0.1.0",
		server.WithToolCapabilities(true),
	)
}

// addToolGroupMCPServer adds or updates the MCP proxy server for a given tool group name.
// If a group with the same name already exists, it will be replaced.
// This method is safe to call concurrently.
func (s *ToolGroupService) addToolGroupMCPServer(name string, mcpServer *server.MCPServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mcpServers[name] = mcpServer
}

// deleteToolGroupMCPServer removes the MCP proxy server for a given tool group name.
func (s *ToolGroupService) deleteToolGroupMCPServer(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.mcpServers, name)
}

// initToolGroupMCPServers initializes the MCP proxy servers for all existing tool groups in the database.
func (s *ToolGroupService) initToolGroupMCPServers() error {
	groups, err := s.ListToolGroups()
	if err != nil {
		return fmt.Errorf("failed to list tool groups from DB: %w", err)
	}
	for _, group := range groups {
		toolNames, err := group.GetTools()
		if err != nil {
			return fmt.Errorf("failed to parse toolNames for group %s: %w", group.Name, err)
		}
		// TODO: Log a warning if a group has no tools, ie, len(toolNames) == 0

		mcpServer := s.newMCPServer(group.Name)
		for _, name := range toolNames {
			tool, exists := s.mcpService.GetToolInstance(name)
			if !exists {
				// it is possible that a tool group contains a tool that does not exist.
				// this should not prevent server startup, so just skip instead of returning an error.
				// TODO: Add a warning log here.
				continue
			}
			mcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
		}
		s.addToolGroupMCPServer(group.Name, mcpServer)
	}
	return nil
}

// handleToolDeletion is a callback that is called when one or more tools is deleted or disabled.
// It removes the tools from all tool group MCP proxy servers.
func (s *ToolGroupService) handleToolDeletion(tools ...string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, mcpServer := range s.mcpServers {
		mcpServer.DeleteTools(tools...)
	}
}
