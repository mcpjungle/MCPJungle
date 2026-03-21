package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

// listResourcesHandler returns a list of all resources, or all resources for a given mcp server if "server" query param is provided
func (s *Server) listResourcesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		server := c.Query("server")
		var (
			resources []model.Resource
			err       error
		)
		if server == "" {
			resources, err = s.mcpService.ListResources()
		} else {
			resources, err = s.mcpService.ListResourcesByServer(server)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resources)
	}
}
