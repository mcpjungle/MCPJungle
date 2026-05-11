package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

type dashboardToggleRequest struct {
	Enabled bool `json:"enabled"`
}

func (s *Server) dashboardRegisterServerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input types.RegisterServerInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		server, err := createServerModelFromInput(&input)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := s.mcpService.RegisterMcpServerWithOAuthSupport(c, &input, server, false, "dashboard"); err != nil {
			var oauthErr *mcp.UpstreamOAuthAuthorizationPendingError
			if errors.As(err, &oauthErr) {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Upstream OAuth authorization is required for this server and is not supported in the dashboard yet.",
				})
				return
			}
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"name":        server.Name,
			"transport":   server.Transport,
			"enabled":     server.Enabled,
			"description": server.Description,
		})
	}
}

func (s *Server) dashboardDeleteServerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := s.mcpService.DeregisterMcpServer(c.Param("name")); err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

func (s *Server) dashboardSetServerEnabledHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input dashboardToggleRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := s.mcpService.SetDashboardServerEnabled(c.Param("name"), input.Enabled); err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"name": c.Param("name"), "enabled": input.Enabled})
	}
}

func (s *Server) dashboardSetToolEnabledHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input dashboardToggleRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		entity := c.Param("name")
		var err error
		if input.Enabled {
			_, err = s.mcpService.EnableTools(entity)
		} else {
			_, err = s.mcpService.DisableTools(entity)
		}
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"name": entity, "enabled": input.Enabled})
	}
}

func (s *Server) dashboardSetPromptEnabledHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input dashboardToggleRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		entity := c.Param("name")
		var err error
		if input.Enabled {
			_, err = s.mcpService.EnablePrompts(entity)
		} else {
			_, err = s.mcpService.DisablePrompts(entity)
		}
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"name": entity, "enabled": input.Enabled})
	}
}
