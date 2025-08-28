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

func NewToolGroupService(db *gorm.DB, mcpService *mcp.MCPService) *ToolGroupService {
	return &ToolGroupService{
		db:         db,
		mcpService: mcpService,
		mcpServers: make(map[string]*server.MCPServer),
		mu:         sync.RWMutex{},
	}
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
	mcpServer := server.NewMCPServer(
		fmt.Sprintf("MCPJungle proxy MCP server for tool group: %s", group.Name),
		"0.1.0",
		server.WithToolCapabilities(true),
	)
	// populate the MCP server with the specified tools
	for _, name := range toolNames {
		tool, exists := s.mcpService.GetToolInstance(name)
		if !exists {
			return fmt.Errorf("tool %s does not exist in the tool tracker", name)
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

// addToolGroupMCPServer adds or updates the MCP proxy server for a given tool group name.
// If a group with the same name already exists, it will be replaced.
// This method is safe to call concurrently.
func (s *ToolGroupService) addToolGroupMCPServer(name string, mcpServer *server.MCPServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mcpServers[name] = mcpServer
}
