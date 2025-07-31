package api

import (
	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"net/http"
)

func registerServerHandler(mcpService *mcp.MCPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input types.RegisterServerInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		server := model.McpServer{
			Name:        input.Name,
			URL:         input.URL,
			Description: input.Description,
			BearerToken: input.BearerToken,
		}
		if err := mcpService.RegisterMcpServer(c, &server); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, server)
	}
}

func deregisterServerHandler(mcpService *mcp.MCPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if err := mcpService.DeregisterMcpServer(name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func listServersHandler(mcpService *mcp.MCPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		servers, err := mcpService.ListMcpServers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, servers)
	}
}
