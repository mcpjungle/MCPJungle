package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	lazyQueryKey   = "mode"
	lazyQueryValue = "lazy"
)

func isLazyRequest(c *gin.Context) bool {
	return c.Query(lazyQueryKey) == lazyQueryValue
}

func (s *Server) mcpProxyDispatchHandler(eager http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isLazyRequest(c) {
			c.Request = c.Request.WithContext(s.mcpService.WithLazyGroup(c.Request.Context(), ""))
			s.lazyMcpHandler.ServeHTTP(c.Writer, c.Request)
			return
		}
		eager.ServeHTTP(c.Writer, c.Request)
	}
}

func (s *Server) toolGroupMCPServerDispatchHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		groupName := c.Param("name")
		if isLazyRequest(c) {
			c.Request = c.Request.WithContext(s.mcpService.WithLazyGroup(c.Request.Context(), groupName))
			s.lazyMcpHandler.ServeHTTP(c.Writer, c.Request)
			return
		}
		s.toolGroupMCPServerCallHandler()(c)
	}
}
