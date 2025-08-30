package api

import (
	"encoding/json"
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

		// Construct the endpoint for the created tool group
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		endpointURL := &url.URL{
			Scheme: scheme,
			Host:   c.Request.Host,
			Path:   fmt.Sprintf("%s/groups/%s/mcp", V0PathPrefix, input.Name),
		}
		resp := &types.CreateToolGroupResponse{
			Endpoint: endpointURL.String(),
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if group == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "tool group not found"})
			return
		}

		resp := &types.ToolGroup{
			Name:        group.Name,
			Description: group.Description,
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
		//  The CLI should then handle this and output "group does not exist".
		c.Status(http.StatusNoContent)
	}
}

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
		streamableServer := server.NewStreamableHTTPServer(groupMcpServer)
		streamableServer.ServeHTTP(c.Writer, c.Request)
	}
}
