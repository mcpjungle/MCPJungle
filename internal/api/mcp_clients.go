package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcpclient"
)

func listMcpClientsHandler(mcpClientService *mcpclient.McpClientService) gin.HandlerFunc {
	return func(c *gin.Context) {
		clients, err := mcpClientService.ListClients()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, clients)
	}
}

func createMcpClientHandler(mcpClientService *mcpclient.McpClientService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.McpClient
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		// TODO: if allow list in the request is null, convert it to an empty JSON array
		client, err := mcpClientService.CreateClient(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, client)
	}
}

func deleteMcpClientHandler(mcpClientService *mcpclient.McpClientService) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		if err := mcpClientService.DeleteClient(name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
