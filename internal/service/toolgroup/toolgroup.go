// Package toolgroup provides functionality to manage tool groups and their associated MCP proxy servers.
package toolgroup

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"sync"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/mcpjungle/mcpjungle/pkg/util"
	"github.com/mcpjungle/mcpjungle/pkg/version"
	"gorm.io/gorm"
)

var ErrToolGroupNotFound = fmt.Errorf("tool group not found: %w", apierrors.ErrNotFound)

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
	// key: tool group name, value: MCP proxy server
	mcpServers map[string]*server.MCPServer
	// mcpServersMu protects access to the mcpServers map
	mcpServersMu sync.RWMutex

	// sseMcpServers manages the SSE MCP proxy servers for all the tool groups
	// key: tool group name, value: MCP proxy server
	sseMcpServers map[string]*server.MCPServer
	// sseMcpServerMu protects access to the sseMcpServers map
	sseMcpServerMu sync.RWMutex
}

func NewToolGroupService(db *gorm.DB, mcpService *mcp.MCPService) (*ToolGroupService, error) {
	s := &ToolGroupService{
		db:         db,
		mcpService: mcpService,

		mcpServers:   make(map[string]*server.MCPServer),
		mcpServersMu: sync.RWMutex{},

		sseMcpServers:  make(map[string]*server.MCPServer),
		sseMcpServerMu: sync.RWMutex{},
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
		return fmt.Errorf("tool group name cannot be empty: %w", apierrors.ErrInvalidInput)
	}
	if !ValidGroupName.MatchString(group.Name) {
		return fmt.Errorf(
			"invalid group name: name must start with an alphanumeric character and "+
				"can only contain alphanumeric characters, underscores, and hyphens: %w",
			apierrors.ErrInvalidInput,
		)
	}

	// resolve all effective tools for this group
	toolNames, err := group.ResolveEffectiveTools(s.mcpService)
	if err != nil {
		return fmt.Errorf("failed to resolve effective tools: %w", err)
	}
	if len(toolNames) == 0 {
		return fmt.Errorf(
			"tool group must contain at least one tool after resolving servers and exclusions: %w",
			apierrors.ErrInvalidInput,
		)
	}

	// create the proxy MCP servers that expose only specified tools
	mcpServer := s.newMCPServer(group.Name)
	sseMcpServer := s.newSseMCPServer(group.Name)

	// populate the MCP servers with the specified tools
	// this also has a side effect of validating that the tools exist in mcpjungle.
	// if a tool does not exist, return an error without creating the group.
	for _, name := range toolNames {
		tool, exists := s.mcpService.GetToolInstance(name)
		if !exists {
			return fmt.Errorf("tool %s does not exist or is disabled: %w", name, apierrors.ErrInvalidInput)
		}

		parentServer, err := s.mcpService.GetToolParentServer(name)
		if err != nil {
			return fmt.Errorf("failed to get parent MCP server of the tool %s: %w", name, err)
		}

		if parentServer.Transport == types.TransportSSE {
			sseMcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
		} else {
			mcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
		}
	}

	// first, add the tool group to the database
	// this also checks for uniqueness of the group's name
	if err := s.db.Create(group).Error; err != nil {
		return fmt.Errorf("failed to create tool group: %w", err)
	}

	// finally, add the proxy MCPs to the tool group MCPs manager so that it is ready to serve
	s.addToolGroupMCPServer(group.Name, mcpServer)
	s.addToolGroupSseMCPServer(group.Name, sseMcpServer)

	return nil
}

// UpdateToolGroup updates an existing tool group without causing any downtime for its MCP proxy servers.
// It returns the configuration of the original tool group before the update.
// If the tool group does not exist, it returns ErrToolGroupNotFound.
func (s *ToolGroupService) UpdateToolGroup(name string, updatedGroup *model.ToolGroup) (*model.ToolGroup, error) {
	oldGroup, err := s.GetToolGroup(name)
	if err != nil {
		if errors.Is(err, ErrToolGroupNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to retrieve the tool group: %w", err)
	}

	// determine which tools were added or removed from the group
	oldToolNames, err := oldGroup.ResolveEffectiveTools(s.mcpService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve effective tools of original group: %w", err)
	}
	updatedToolNames, err := updatedGroup.ResolveEffectiveTools(s.mcpService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve effective tools of the updated group: %w", err)
	}

	toolsAdded, toolsRemoved := util.DiffTools(oldToolNames, updatedToolNames)

	// if nothing was actually changed in the group, no need to proceed further
	if updatedGroup.Description == oldGroup.Description && len(toolsAdded) == 0 && len(toolsRemoved) == 0 {
		return oldGroup, nil
	}

	// determine the changes to make to the tool group's proxy MCP server instances (normal + SSE)
	// all changes are ultimately made at the end of this method to avoid inconsistent state in case of errors.
	mcpServer, exists := s.GetToolGroupMCPServer(name)
	if !exists {
		return nil, fmt.Errorf("MCP server for tool group %s does not exist", name)
	}
	sseMcpServer, exists := s.GetToolGroupSseMCPServer(name)
	if !exists {
		return nil, fmt.Errorf("SSE MCP server for tool group %s does not exist", name)
	}

	// tools added to the group must be added to its MCP server instances
	var sseToolsToAdd, normalToolsToAdd []mcpgo.Tool
	for _, toolName := range toolsAdded {
		tool, exists := s.mcpService.GetToolInstance(toolName)
		if !exists {
			return nil, fmt.Errorf("tool %s does not exist or is disabled: %w", toolName, apierrors.ErrInvalidInput)
		}

		parentServer, err := s.mcpService.GetToolParentServer(toolName)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent MCP server of the tool %s: %w", toolName, err)
		}

		if parentServer.Transport == types.TransportSSE {
			sseToolsToAdd = append(sseToolsToAdd, tool)
		} else {
			normalToolsToAdd = append(normalToolsToAdd, tool)
		}
	}

	// tools removed from the group must be removed from its MCP server instances
	//
	// make all the changes together to avoid inconsistent state in case of errors.
	// DeleteTools is safe to call against both proxies for every removed tool:
	// each canonical tool name can only exist in one proxy, and deleting a
	// non-existent tool from the other proxy is a no-op.
	mcpServer.DeleteTools(toolsRemoved...)
	sseMcpServer.DeleteTools(toolsRemoved...)

	for _, tool := range normalToolsToAdd {
		mcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
	}
	for _, tool := range sseToolsToAdd {
		sseMcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
	}

	// as a final step, update the tool group record in the database
	// we only persist this update after successfully updating the in-memory state

	// ensure the group name remains unchanged in the db record
	updatedGroup.Name = name
	if err := s.db.Model(&model.ToolGroup{}).Where("name = ?", name).Updates(updatedGroup).Error; err != nil {
		return nil, fmt.Errorf("failed to update tool group in DB: %w", err)
	}

	return oldGroup, nil
}

// ResolveEffectiveTools resolves all effective tools for the specified tool group.
// The resulting list is sorted for deterministic API responses and tests.
func (s *ToolGroupService) ResolveEffectiveTools(name string) ([]string, error) {
	group, err := s.GetToolGroup(name)
	if err != nil {
		return nil, err
	}

	tools, err := group.ResolveEffectiveTools(s.mcpService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve effective tools for group %s: %w", name, err)
	}

	sort.Strings(tools)
	return tools, nil
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
	s.deleteToolGroupMCPServers(name)

	err := s.db.Unscoped().Where("name = ?", name).Delete(&model.ToolGroup{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete toolgroup: %w", err)
	}
	return nil
}

// GetToolGroupMCPServer retrieves the MCP proxy server for a given tool group name.
func (s *ToolGroupService) GetToolGroupMCPServer(name string) (*server.MCPServer, bool) {
	s.mcpServersMu.RLock()
	defer s.mcpServersMu.RUnlock()
	mcpServer, exists := s.mcpServers[name]
	return mcpServer, exists
}

// GetToolGroupSseMCPServer retrieves the SSE MCP proxy server for a given tool group name.
func (s *ToolGroupService) GetToolGroupSseMCPServer(name string) (*server.MCPServer, bool) {
	s.sseMcpServerMu.RLock()
	defer s.sseMcpServerMu.RUnlock()
	mcpServer, exists := s.sseMcpServers[name]
	return mcpServer, exists
}

// newMCPServer creates a new MCP proxy server for a given tool group name.
// The advertised version is tied to the mcpjungle server version (pkg/version)
// so each tool-group proxy reports the host's version instead of a hardcoded
// string.
func (s *ToolGroupService) newMCPServer(groupName string) *server.MCPServer {
	return server.NewMCPServer(
		fmt.Sprintf("MCPJungle proxy MCP server for tool group: %s", groupName),
		version.GetVersion(),
		server.WithResourceCapabilities(false, false),
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithToolFilter(mcp.ProxyToolFilter),
	)
}

// newSseMCPServer creates a new SSE MCP proxy server for a given tool group name.
func (s *ToolGroupService) newSseMCPServer(groupName string) *server.MCPServer {
	return server.NewMCPServer(
		fmt.Sprintf("MCPJungle proxy MCP server for SSE transport for tool group: %s", groupName),
		version.GetVersion(),
		server.WithResourceCapabilities(false, false),
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithToolFilter(mcp.ProxyToolFilter),
	)
}

// addToolGroupMCPServer adds or updates the MCP proxy server for a given tool group name.
// If a group with the same name already exists, it will be replaced.
// This method is safe to call concurrently.
func (s *ToolGroupService) addToolGroupMCPServer(name string, mcpServer *server.MCPServer) {
	s.mcpServersMu.Lock()
	defer s.mcpServersMu.Unlock()
	s.mcpServers[name] = mcpServer
}

// addToolGroupSseMCPServer adds or updates the SSE MCP proxy server for a given tool group name.
// If a group with the same name already exists, it will be replaced.
// This method is safe to call concurrently.
func (s *ToolGroupService) addToolGroupSseMCPServer(name string, mcpServer *server.MCPServer) {
	s.sseMcpServerMu.Lock()
	defer s.sseMcpServerMu.Unlock()
	s.sseMcpServers[name] = mcpServer
}

// deleteToolGroupMCPServers removes the MCP proxy servers for a given tool group name.
func (s *ToolGroupService) deleteToolGroupMCPServers(name string) {
	// first, acquire both locks to ensure complete cleanup of the group
	s.mcpServersMu.Lock()
	defer s.mcpServersMu.Unlock()

	s.sseMcpServerMu.Lock()
	defer s.sseMcpServerMu.Unlock()

	// proceed to delete both normal & sse proxies for the group, then release the locks
	delete(s.mcpServers, name)
	delete(s.sseMcpServers, name)
}

// initToolGroupMCPServers initializes the MCP proxy servers for all existing tool groups in the database.
// It initializes both the mcpServers and sseMcpServers.
func (s *ToolGroupService) initToolGroupMCPServers() error {
	groups, err := s.ListToolGroups()
	if err != nil {
		return fmt.Errorf("failed to list tool groups from DB: %w", err)
	}

	for _, group := range groups {
		mcpServer := s.newMCPServer(group.Name)
		sseMcpServer := s.newSseMCPServer(group.Name)

		toolNames, err := group.ResolveEffectiveTools(s.mcpService)
		if err != nil {
			// If resolution of any server or specific tool fails for a group, the error is logged and
			// the group is added as an empty MCP server to the proxy.
			// This is not ideal, but it ensures that mcpjungle server startup doesn't fail.
			// TODO: Change design to include all other servers & tools that were resolved.
			// See https://github.com/mcpjungle/MCPJungle/issues/233
			log.Printf(
				"[ERROR] failed to resolve effective tools for tool group %s during startup; the tool group will be initialized as empty: %v",
				group.Name,
				err,
			)
			s.addToolGroupMCPServer(group.Name, mcpServer)
			s.addToolGroupSseMCPServer(group.Name, sseMcpServer)
			continue
		}

		// figure out which tools from this group are no longer available and warn about them
		availableTools, missingTools := s.partitionExistingTools(toolNames)
		s.warnAboutMissingGroupTools(group.Name, missingTools)

		for _, name := range availableTools {
			// no need to check for existence because availableTools only contains tools that are available
			tool, _ := s.mcpService.GetToolInstance(name)

			parentServer, err := s.mcpService.GetToolParentServer(name)
			if err != nil {
				return fmt.Errorf("failed to get parent MCP server of the tool %s: %w", name, err)
			}

			if parentServer.Transport == types.TransportSSE {
				sseMcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
			} else {
				mcpServer.AddTool(tool, s.mcpService.MCPProxyToolCallHandler)
			}
		}

		s.addToolGroupMCPServer(group.Name, mcpServer)
		s.addToolGroupSseMCPServer(group.Name, sseMcpServer)
	}

	return nil
}

// handleToolDeletion is a callback that is called when one or more tools is deleted or disabled.
// It removes the tools from all tool group MCP proxy servers.
func (s *ToolGroupService) handleToolDeletion(tools ...string) {
	s.warnGroupsReferencingDeletedTools(tools...)

	s.mcpServersMu.RLock()
	defer s.mcpServersMu.RUnlock()

	s.sseMcpServerMu.Lock()
	defer s.sseMcpServerMu.Unlock()

	for _, mcpServer := range s.mcpServers {
		mcpServer.DeleteTools(tools...)
	}

	for _, sseMcpServer := range s.sseMcpServers {
		sseMcpServer.DeleteTools(tools...)
	}
}

// partitionExistingTools splits a resolved tool-group tool list into:
// - tools that currently exist as live MCPJungle tool instances
// - tools that are still referenced by desired state but are not currently available
//
// Tool groups preserve those missing references in the DB, but only currently
// available tools are exposed through the runtime group proxies.
func (s *ToolGroupService) partitionExistingTools(toolNames []string) ([]string, []string) {
	available := make([]string, 0, len(toolNames))
	missing := make([]string, 0)

	for _, toolName := range toolNames {
		if _, exists := s.mcpService.GetToolInstance(toolName); exists {
			available = append(available, toolName)
			continue
		}
		missing = append(missing, toolName)
	}

	return available, missing
}

// warnAboutMissingGroupTools emits a runtime warning when a tool group's
// desired state references tools that do not currently exist in MCPJungle.
//
// This warning-only behavior is intentional: MCPJungle does not mutate the
// persisted tool-group config, and the group proxy simply omits the missing
// tools until they reappear or the user updates the config.
func (s *ToolGroupService) warnAboutMissingGroupTools(groupName string, missingTools []string) {
	if len(missingTools) == 0 {
		return
	}
	log.Printf(
		"[WARN] tool group '%s' references tools that do not currently exist in mcpjungle or are disabled: %v. Update the tool group configuration to make this warning go away.",
		groupName,
		missingTools,
	)
}

// warnGroupsReferencingDeletedTools inspects persisted tool-group configs after
// one or more tools disappear from MCPJungle and logs warnings for any stale
// references that remain in desired state.
//
// This keeps operators informed without rewriting user-authored group
// definitions when upstream tool availability changes.
func (s *ToolGroupService) warnGroupsReferencingDeletedTools(toolNames ...string) {
	if len(toolNames) == 0 {
		return
	}

	groups, err := s.ListToolGroups()
	if err != nil {
		log.Printf("[WARN] failed to list tool groups while checking for stale tool references: %v", err)
		return
	}

	// prepare a set of deleted tool names for efficient lookup when inspecting groups
	deleted := make(map[string]struct{}, len(toolNames))
	for _, name := range toolNames {
		deleted[name] = struct{}{}
	}

	for i := range groups {
		referenced := make([]string, 0)

		// only examine tools that are explicitly mentioned in the group config.
		includedTools, err := groups[i].GetTools()
		if err != nil {
			log.Printf("[WARN] failed to inspect included tools for tool group '%s': %v", groups[i].Name, err)
			continue
		}
		for _, toolName := range includedTools {
			if _, exists := deleted[toolName]; exists {
				referenced = append(referenced, toolName)
			}
		}

		excludedTools, err := groups[i].GetExcludedTools()
		if err != nil {
			log.Printf("[WARN] failed to inspect excluded tools for tool group '%s': %v", groups[i].Name, err)
			continue
		}
		for _, toolName := range excludedTools {
			if _, exists := deleted[toolName]; exists {
				referenced = append(referenced, toolName)
			}
		}

		s.warnAboutMissingGroupTools(groups[i].Name, referenced)
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
		groupTools, err := groups[i].ResolveEffectiveTools(s.mcpService)
		if err != nil {
			return fmt.Errorf("failed to resolve effective tools for group %s: %w", name, err)
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

	parentServer, err := s.mcpService.GetToolParentServer(newTool)
	if err != nil {
		return fmt.Errorf("failed to get parent MCP server of the tool %s: %w", newTool, err)
	}

	// add the new tool instance to all relevant MCP proxy servers
	s.mcpServersMu.RLock()
	defer s.mcpServersMu.RUnlock()

	s.sseMcpServerMu.Lock()
	defer s.sseMcpServerMu.Unlock()

	for _, name := range groupsToUpdate {
		if parentServer.Transport == types.TransportSSE {
			sseMcpServer, exists := s.sseMcpServers[name]
			if exists {
				sseMcpServer.AddTool(newToolInstance, s.mcpService.MCPProxyToolCallHandler)
			}
			continue
		}

		mcpServer, exists := s.mcpServers[name]
		if exists {
			mcpServer.AddTool(newToolInstance, s.mcpService.MCPProxyToolCallHandler)
		}
	}

	return nil
}
