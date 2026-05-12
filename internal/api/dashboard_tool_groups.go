package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

type dashboardToolGroupCreateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
}

type dashboardToolGroupTool struct {
	Name          string `json:"name"`
	CanonicalName string `json:"canonical_name"`
	Server        string `json:"server"`
	Description   string `json:"description,omitempty"`
}

type dashboardToolGroup struct {
	Name                   string                   `json:"name"`
	Description            string                   `json:"description,omitempty"`
	ToolCount              int                      `json:"tool_count"`
	Tools                  []dashboardToolGroupTool `json:"tools"`
	StreamableHTTPEndpoint string                   `json:"streamable_http_endpoint"`
	SSEEndpoint            string                   `json:"sse_endpoint"`
	SSEMessageEndpoint     string                   `json:"sse_message_endpoint"`
}

type dashboardToolGroupsResponse struct {
	ToolGroups []dashboardToolGroup       `json:"tool_groups"`
	EmptyState *types.DashboardEmptyState `json:"empty_state,omitempty"`
}

func (s *Server) dashboardToolGroupsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		groups, err := s.toolGroupService.ListToolGroups()
		if err != nil {
			handleServiceError(c, err)
			return
		}

		resp := dashboardToolGroupsResponse{
			ToolGroups: make([]dashboardToolGroup, 0, len(groups)),
		}
		for _, group := range groups {
			item, err := s.buildDashboardToolGroup(c, group)
			if err != nil {
				handleServiceError(c, err)
				return
			}
			resp.ToolGroups = append(resp.ToolGroups, item)
		}

		if len(resp.ToolGroups) == 0 {
			resp.EmptyState = &types.DashboardEmptyState{
				Title:       "No tool groups configured yet.",
				Description: "Create a tool group to expose a focused subset of MCP tools.",
				Commands: []string{
					"mcpjungle create group --conf group.json",
					"mcpjungle list groups",
					"mcpjungle get group <group-name>",
				},
			}
		}

		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) dashboardGetToolGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		group, err := s.toolGroupService.GetToolGroup(c.Param("name"))
		if err != nil {
			handleServiceError(c, err)
			return
		}

		resp, err := s.buildDashboardToolGroup(c, *group)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) dashboardCreateToolGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input dashboardToolGroupCreateRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		includedTools, err := json.Marshal(input.Tools)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tools payload"})
			return
		}

		group := &model.ToolGroup{
			Name:          input.Name,
			Description:   input.Description,
			IncludedTools: includedTools,
		}
		if err := s.toolGroupService.CreateToolGroup(group); err != nil {
			handleServiceError(c, err)
			return
		}

		resp, err := s.buildDashboardToolGroup(c, *group)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, resp)
	}
}

func (s *Server) dashboardDeleteToolGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := s.toolGroupService.DeleteToolGroup(c.Param("name")); err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

func (s *Server) buildDashboardToolGroup(c *gin.Context, group model.ToolGroup) (dashboardToolGroup, error) {
	toolNames, err := group.ResolveEffectiveTools(s.mcpService)
	if err != nil {
		return dashboardToolGroup{}, err
	}

	tools := make([]dashboardToolGroupTool, 0, len(toolNames))
	for _, toolName := range toolNames {
		item := dashboardToolGroupTool{
			CanonicalName: toolName,
			Name:          toolName,
			Server:        "Unknown",
		}
		if tool, err := s.mcpService.GetTool(toolName); err == nil {
			item.Name = tool.Name
			if server, serverErr := s.mcpService.GetToolParentServer(toolName); serverErr == nil {
				item.Server = server.Name
			}
			item.CanonicalName = toolName
			item.Description = tool.Description
		}
		respName := item.Name
		parts := strings.SplitN(toolName, "__", 2)
		if len(parts) == 2 {
			respName = parts[1]
		}
		item.Name = respName
		tools = append(tools, item)
	}

	endpoints := getToolGroupEndpoints(c, group.Name)

	return dashboardToolGroup{
		Name:                   group.Name,
		Description:            group.Description,
		ToolCount:              len(tools),
		Tools:                  tools,
		StreamableHTTPEndpoint: endpoints.StreamableHTTPEndpoint,
		SSEEndpoint:            endpoints.SSEEndpoint,
		SSEMessageEndpoint:     endpoints.SSEMessageEndpoint,
	}, nil
}
