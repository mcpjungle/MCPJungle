package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
)

func listToolsHandler(mcpService *mcp.MCPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		server := c.Query("server")
		var (
			tools []model.Tool
			err   error
		)
		if server == "" {
			// no server specified, list all tools
			tools, err = mcpService.ListTools()
		} else {
			// server specified, list tools for that server
			tools, err = mcpService.ListToolsByServer(server)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, tools)
	}
}

// invokeToolHandler forwards the JSON body to the tool URL and streams response back.
func invokeToolHandler(mcpService *mcp.MCPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var args map[string]any
		if err := json.NewDecoder(c.Request.Body).Decode(&args); err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "failed to decode request body: " + err.Error()},
			)
			return
		}

		rawName, ok := args["name"]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'name' field in request body"})
			return
		}
		name, ok := rawName.(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "'name' field must be a string"})
			return
		}

		// remove name from args since it was an input for the api, not for the tool
		delete(args, "name")

		resp, err := mcpService.InvokeTool(c, name, args)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to invoke tool: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

// getToolHandler returns the tool with the given name.
func getToolHandler(mcpService *mcp.MCPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// tool name has to be supplied as a query param because it contains slash.
		// cannot be supplied as a path param.
		name := c.Query("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'name' query parameter"})
			return
		}
		tool, err := mcpService.GetTool(name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get tool: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, tool)
	}
}

// enableToolsHandler enables the given tool or all tools of the given mcp server
func enableToolsHandler(mcpService *mcp.MCPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		entity := c.Query("entity")
		if entity == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'entity' query parameter"})
			return
		}
		enabledTools, err := mcpService.EnableTools(entity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable tool(s): " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, enabledTools)
	}
}

// disableToolsHandler disables the given tool or all tools of the given mcp server
func disableToolsHandler(mcpService *mcp.MCPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		entity := c.Query("entity")
		if entity == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'entity' query parameter"})
			return
		}
		disabledTools, err := mcpService.DisableTools(entity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable tool(s): " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, disabledTools)
	}
}
