package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/toolgroup"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"net/http"
	"net/url"
)

func createToolGroupHandler(toolGroupService *toolgroup.ToolGroupService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input model.ToolGroup
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := toolGroupService.CreateToolGroup(&input); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		resp := &types.CreateToolGroupResponse{
			Endpoint: getToolGroupEndpoint(c, input.Name),
		}
		c.JSON(http.StatusCreated, resp)
	}
}

// listToolGroupsHandler handles returns a list of all tool groups.
// This API only provides basic information about each tool group, ie, name and description.
func listToolGroupsHandler(toolGroupService *toolgroup.ToolGroupService) gin.HandlerFunc {
	return func(c *gin.Context) {
		groups, err := toolGroupService.ListToolGroups()
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
		}

		c.JSON(http.StatusOK, resp)
	}
}

func getToolGroupHandler(toolGroupService *toolgroup.ToolGroupService) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		group, err := toolGroupService.GetToolGroup(name)
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
			Endpoint: getToolGroupEndpoint(c, group.Name),
		}
		// Convert datatypes.JSON to []string
		if group.IncludedTools != nil {
			var tools []string
			if err := json.Unmarshal(group.IncludedTools, &tools); err != nil {
				// TODO: Log error or handle it appropriately
				tools = []string{}
			}
			resp.IncludedTools = tools
		}

		c.JSON(http.StatusOK, resp)
	}
}

func deleteToolGroupHandler(toolGroupService *toolgroup.ToolGroupService) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		err := toolGroupService.DeleteToolGroup(name)
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

// toolGroupMCPServerCallHandler handles incoming MCP requests from for a specific tool group.
func toolGroupMCPServerCallHandler(toolGroupService *toolgroup.ToolGroupService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// get the Proxy MCP server for the specified tool group
		groupName := c.Param("name")
		groupMcpServer, exists := toolGroupService.GetToolGroupMCPServer(groupName)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("tool group not found: %s", groupName)})
			return
		}

		// serve the MCP request using the MCP server
		// TODO: Make this API more efficient
		// This api sits in the host path because we expect high traffic on MCP tool calling.
		// It is inefficient to create a new StreamableHTTPServer for each request.
		// Maybe pre-create a StreamableHTTPServer for each tool group and store it in the ToolGroupMCPServer struct?
		streamableServer := server.NewStreamableHTTPServer(groupMcpServer)
		streamableServer.ServeHTTP(c.Writer, c.Request)
	}
}

// getToolGroupEndpoint deduces the proxy MCP server endpoint URL for a given tool group
func getToolGroupEndpoint(c *gin.Context, groupName string) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	endpointURL := &url.URL{
		Scheme: scheme,
		Host:   c.Request.Host,
		Path:   fmt.Sprintf("%s/groups/%s/mcp", V0PathPrefix, groupName),
	}
	return endpointURL.String()
}
