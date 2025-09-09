// Package toolgroup provides functionality to manage tool groups and their associated MCP proxy servers.
package toolgroup

import (
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"gorm.io/gorm"
)

var ErrToolGroupNotFound = errors.New("tool group not found")

// ValidGroupName is a regex that matches valid tool group names.
// A valid tool group name must start with an alphanumeric character and can contain
// alphanumeric characters, underscores, and hyphens.
// This ensures that the group name can be safely used in URLs.
var ValidGroupName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

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

	// register callbacks with mcp service to be notified when a tool gets added/removed
	mcpService.SetToolDeletionCallback(s.handleToolDeletion)
	mcpService.SetToolAdditionCallback(s.handleToolAddition)

	if err := s.initToolGroupMCPServers(); err != nil {
		return nil, fmt.Errorf("failed to initialize tool group MCP servers: %w", err)
	}
	return s, nil
}

// CreateToolGroup creates a new tool group in the database and a Proxy MCP server that just exposes the specified tools.
func (s *ToolGroupService) CreateToolGroup(group *model.ToolGroup) error {
	// validate the tool group name
	if len(group.Name) == 0 {
		return errors.New("tool group name cannot be empty")
	}
	if !ValidGroupName.MatchString(group.Name) {
		return fmt.Errorf(
			"invalid group name: name must start with an alphanumeric character and " +
				"can only contain alphanumeric characters, underscores, and hyphens",
		)
	}

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
	// this also has a side effect of validating that the tools exist in mcpjungle.
	// if a tool does not exist, return an error without creating the group.
	for _, name := range toolNames {
		tool, exists := s.mcpService.GetToolInstance(name)
		if !exists {
			return fmt.Errorf("tool %s does not exist or is disabled", name)
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
			return nil, ErrToolGroupNotFound
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

// handleToolAddition is a callback that is called when a tool is added or (re)enabled in mcpjungle.
// this callback adds the new tool to MCP proxy servers of all groups that include it.
func (s *ToolGroupService) handleToolAddition(newTool string) error {
	// get all tool groups from the database
	groups, err := s.ListToolGroups()
	if err != nil {
		return fmt.Errorf("failed to list tool groups from DB: %w", err)
	}

	// find all groups that include the added tool
	groupsToUpdate := make([]string, 0, len(groups))
	for i := range groups {
		name := groups[i].Name
		groupTools, err := groups[i].GetTools()
		if err != nil {
			return fmt.Errorf("failed to get tool names for group %s: %w", name, err)
		}
		for _, t := range groupTools {
			if t != newTool {
				continue
			}
			// current group includes the added tool, so add the tool instance to the group's MCP server
			groupsToUpdate = append(groupsToUpdate, name)
			// no need to check other tools in this group anymore, so exit the loop and move on to the next group
			break
		}
	}

	newToolInstance, exists := s.mcpService.GetToolInstance(newTool)
	if !exists {
		// this should not happen because the tool should exist if we are in this callback
		return fmt.Errorf("tool instance %s does not exist", newTool)
	}

	// add the new tool instance to all relevant MCP proxy servers
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, name := range groupsToUpdate {
		mcpServer, exists := s.mcpServers[name]
		if exists {
			mcpServer.AddTool(newToolInstance, s.mcpService.MCPProxyToolCallHandler)
		}
	}

	return nil
}
