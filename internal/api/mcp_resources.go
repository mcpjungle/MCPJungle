package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
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
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, resources)
	}
}

// getResourceHandler returns resource metadata for the given URI.
func (s *Server) getResourceHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request types.ResourceGetRequest
		if err := json.NewDecoder(c.Request.Body).Decode(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to decode request body: " + err.Error()})
			return
		}

		if request.URI == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'uri' field in request body"})
			return
		}

		resource, err := s.mcpService.GetResource(request.URI)
		if err != nil {
			handleServiceError(c, fmt.Errorf("failed to get resource: %w", err))
			return
		}

		c.JSON(http.StatusOK, resource)
	}
}

// readResourceHandler reads live resource content for the given URI.
func (s *Server) readResourceHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request types.ResourceReadRequest
		if err := json.NewDecoder(c.Request.Body).Decode(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to decode request body: " + err.Error()})
			return
		}

		if request.URI == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'uri' field in request body"})
			return
		}

		resp, err := s.mcpService.ReadResource(c, request.URI)
		if err != nil {
			handleServiceError(c, fmt.Errorf("failed to read resource: %w", err))
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}
