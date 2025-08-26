package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mcpjungle/mcpjungle/internal/model"
)

func (s *Server) listMcpClientsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clients, err := s.mcpClientService.ListClients()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, clients)
	}
}

func (s *Server) createMcpClientHandler() gin.HandlerFunc {
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
		client, err := s.mcpClientService.CreateClient(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, client)
	}
}

func (s *Server) deleteMcpClientHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		if err := s.mcpClientService.DeleteClient(name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
