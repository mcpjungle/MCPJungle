package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/toolgroup"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func (s *Server) createToolGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input model.ToolGroup
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := s.toolGroupService.CreateToolGroup(&input); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		resp := &types.CreateToolGroupResponse{
			ToolGroupEndpoints: getToolGroupEndpoints(c, input.Name),
		}
		c.JSON(http.StatusCreated, resp)
	}
}

// listToolGroupsHandler handles returns a list of all tool groups.
// This API only provides basic information about each tool group, ie, name and description.
func (s *Server) listToolGroupsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		groups, err := s.toolGroupService.ListToolGroups()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := make([]*types.ToolGroup, len(groups))
		for i, g := range groups {
			resp[i] = &types.ToolGroup{
				Name:        g.Name,
				Description: g.Description,
			}

			gTools, err := g.GetTools()
			if err != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{"error": fmt.Sprintf("error getting included tools of group %s: %s", g.Name, err.Error())},
				)
				return
			}
			resp[i].IncludedTools = gTools

			gServers, err := g.GetServers()
			if err != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{"error": fmt.Sprintf("error getting included servers of group %s: %s", g.Name, err.Error())},
				)
				return
			}
			resp[i].IncludedServers = gServers

			gExcluded, err := g.GetExcludedTools()
			if err != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{"error": fmt.Sprintf("error getting excluded tools of group %s: %s", g.Name, err.Error())},
				)
				return
			}
			resp[i].ExcludedTools = gExcluded
		}

		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) getToolGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		group, err := s.toolGroupService.GetToolGroup(name)
		if err != nil {
			if errors.Is(err, toolgroup.ErrToolGroupNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("tool group %s not found", name)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := &types.GetToolGroupResponse{
			ToolGroup: &types.ToolGroup{
				Name:        group.Name,
				Description: group.Description,
			},
			ToolGroupEndpoints: getToolGroupEndpoints(c, group.Name),
		}

		// Get included tools
		var tools []string
		tools, err = group.GetTools()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("error getting included tools of group: %s", err.Error())},
			)
			return
		}
		resp.IncludedTools = tools

		// Get included servers
		var servers []string
		servers, err = group.GetServers()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("error getting included servers of group: %s", err.Error())},
			)
			return
		}
		resp.IncludedServers = servers

		// Get excluded tools
		var excludedTools []string
		excludedTools, err = group.GetExcludedTools()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("error getting excluded tools of group: %s", err.Error())},
			)
			return
		}
		resp.ExcludedTools = excludedTools

		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) getToolGroupEffectiveToolsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		tools, err := s.toolGroupService.ResolveEffectiveTools(name)
		if err != nil {
			if errors.Is(err, toolgroup.ErrToolGroupNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("tool group %s not found", name)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"tools": tools})
	}
}

func (s *Server) deleteToolGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		err := s.toolGroupService.DeleteToolGroup(name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// TODO: return 404 if the group did not exist.
		//  The tool group service should return ErrToolGroupNotFound if the group does not exist.
		//  The CLI should then handle this and output "group does not exist".
		c.Status(http.StatusNoContent)
	}
}

func (s *Server) updateToolGroupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "group name is required"})
			return
		}

		var input model.ToolGroup
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		originalConf, err := s.toolGroupService.UpdateToolGroup(name, &input)
		if err != nil {
			if errors.Is(err, toolgroup.ErrToolGroupNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("tool group %s does not exist", name)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// create and send response object
		resp := &types.UpdateToolGroupResponse{
			Name: name,
			Old: &types.ToolGroup{
				Name:        originalConf.Name,
				Description: originalConf.Description,
			},
			New: &types.ToolGroup{
				Name:        input.Name,
				Description: input.Description,
			},
		}

		var origTools []string
		origTools, err = originalConf.GetTools()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("error getting included tools of the original group config: %s", err.Error())},
			)
			return
		}
		resp.Old.IncludedTools = origTools

		var origServers []string
		origServers, err = originalConf.GetServers()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("error getting included servers of the original group config: %s", err.Error())},
			)
			return
		}
		resp.Old.IncludedServers = origServers

		var origExcluded []string
		origExcluded, err = originalConf.GetExcludedTools()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("error getting excluded tools of the original group config: %s", err.Error())},
			)
			return
		}
		resp.Old.ExcludedTools = origExcluded

		var newTools []string
		newTools, err = input.GetTools()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("error getting included tools of the new group config: %s", err.Error())},
			)
			return
		}
		resp.New.IncludedTools = newTools

		var newServers []string
		newServers, err = input.GetServers()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("error getting included servers of the new group config: %s", err.Error())},
			)
			return
		}
		resp.New.IncludedServers = newServers

		var newExcluded []string
		newExcluded, err = input.GetExcludedTools()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("error getting excluded tools of the new group config: %s", err.Error())},
			)
			return
		}
		resp.New.ExcludedTools = newExcluded

		c.JSON(http.StatusOK, resp)
	}
}

// toolGroupMCPServerCallHandler handles incoming MCP requests from for a specific tool group.
func (s *Server) toolGroupMCPServerCallHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// get the Proxy MCP server for the specified tool group
		groupName := c.Param("name")
		groupMcpServer, exists := s.toolGroupService.GetToolGroupMCPServer(groupName)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("tool group not found: %s", groupName)})
			return
		}

		// serve the MCP request using the MCP server
		// TODO: Make this API more efficient
		// This api sits in the hot path because we expect high traffic on MCP tool calling.
		// It is inefficient to create a new StreamableHTTPServer for each request.
		// Maybe pre-create a StreamableHTTPServer for each tool group and store it in the ToolGroupMCPServer struct?
		streamableServer := server.NewStreamableHTTPServer(groupMcpServer)
		streamableServer.ServeHTTP(c.Writer, c.Request)
	}
}

// getGroupSseServer returns a server.SSEServer for a specific group, creating one if it doesn't already exist.
// It ensures that each tool group has its own SSE server with the correct dynamic base path.
func (s *Server) getGroupSseServer(groupName string) (*server.SSEServer, error) {
	// Try to get existing server first
	if serverVal, ok := s.groupSseServers.Load(groupName); ok {
		return serverVal.(*server.SSEServer), nil
	}

	// Get the sse MCP proxy server for the group
	groupSseMcpServer, exists := s.toolGroupService.GetToolGroupSseMCPServer(groupName)
	if !exists {
		return nil, fmt.Errorf("tool group not found: %s", groupName)
	}

	// Create new server with the correct dynamic base path
	sseServer := server.NewSSEServer(
		groupSseMcpServer,
		server.WithDynamicBasePath(func(r *http.Request, sessionID string) string {
			// Return the group-specific base path
			return fmt.Sprintf("%s/groups/%s", V0PathPrefix, groupName)
		}),
	)

	// Store for future use
	s.groupSseServers.Store(groupName, sseServer)

	return sseServer, nil
}

// toolGroupSseMCPServerCallHandler handles SSE connection requests (/sse) for a specific tool group.
func (s *Server) toolGroupSseMCPServerCallHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		groupName := c.Param("name")

		groupSseMcpServer, err := s.getGroupSseServer(groupName)
		if err != nil {
			c.JSON(
				http.StatusNotFound,
				gin.H{"error": fmt.Sprintf("failed to get sse server for group %s: %v", groupName, err)},
			)
			return
		}

		groupSseMcpServer.SSEHandler().ServeHTTP(c.Writer, c.Request)
	}
}

// toolGroupSseMCPServerCallHandler handles SSE connection requests (/message) for a specific tool group.
func (s *Server) toolGroupSseMCPServerCallMessageHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		groupName := c.Param("name")

		groupSseMcpServer, err := s.getGroupSseServer(groupName)
		if err != nil {
			c.JSON(
				http.StatusNotFound,
				gin.H{"error": fmt.Sprintf("failed to get sse server for group: %s", groupName)},
			)
			return
		}

		groupSseMcpServer.MessageHandler().ServeHTTP(c.Writer, c.Request)
	}
}

// getToolGroupEndpoints deduces the proxy MCP server endpoint URLs for a given tool group.
// It returns the streamable HTTP endpoint and the SSE endpoints
func getToolGroupEndpoints(c *gin.Context, groupName string) *types.ToolGroupEndpoints {
	// This logic of creating the API endpoints is duplicated from internal/api/server.go
	// TODO: centralize this logic into one place and use that everywhere.
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	endpointURL := &url.URL{
		Scheme: scheme,
		Host:   c.Request.Host,
		Path:   fmt.Sprintf("%s/groups/%s", V0PathPrefix, groupName),
	}
	baseEndpoint := endpointURL.String()

	return &types.ToolGroupEndpoints{
		StreamableHTTPEndpoint: baseEndpoint + "/mcp",
		SSEEndpoint:            baseEndpoint + "/sse",
		SSEMessageEndpoint:     baseEndpoint + "/message",
	}
}
