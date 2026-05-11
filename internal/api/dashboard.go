package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

func (s *Server) dashboardOverviewHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode := c.MustGet("mode").(model.ServerMode)
		resp, err := s.dashboardService.Overview(mode, requestBaseURL(c))
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) dashboardServersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := s.dashboardService.Servers()
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) dashboardToolsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := s.dashboardService.Tools()
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) dashboardPromptsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := s.dashboardService.Prompts()
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) dashboardResourcesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := s.dashboardService.Resources()
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (s *Server) dashboardDiagnosticsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode := c.MustGet("mode").(model.ServerMode)
		resp, err := s.dashboardService.Diagnostics(mode, requestBaseURL(c))
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func requestBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}
