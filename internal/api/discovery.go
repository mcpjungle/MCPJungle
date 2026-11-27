package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// discoveryModeQueryKey is the MCP endpoint query parameter that selects the
	// proxy behavior for a single request.
	discoveryModeQueryKey = "mode"

	// discoveryModeQueryValue is the value that routes a request to the
	// discovery-mode MCP server.
	discoveryModeQueryValue = "discovery"
)

// isDiscoveryModeRequest reports whether the client asked for discovery mode on
// the MCP proxy endpoint.
func isDiscoveryModeRequest(c *gin.Context) bool {
	return c.Query(discoveryModeQueryKey) == discoveryModeQueryValue
}

// mcpProxyDispatchHandler routes MCP proxy requests to the discovery-mode server
// when the client opts in with the discovery query parameter, and to the eager
// proxy otherwise. Authentication middleware runs before this handler, so both
// modes share the same client identity and request context.
func (s *Server) mcpProxyDispatchHandler(eager http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isDiscoveryModeRequest(c) && s.discoveryMcpHandler != nil {
			s.discoveryMcpHandler.ServeHTTP(c.Writer, c.Request)
			return
		}
		eager.ServeHTTP(c.Writer, c.Request)
	}
}

// sseProxyDispatchHandler routes legacy SSE connection setup to the
// discovery-mode SSE server when the client opts in with the discovery query
// parameter. Discovery sessions advertise their own message endpoint, so
// message routing never mixes the two servers.
func (s *Server) sseProxyDispatchHandler(eager http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isDiscoveryModeRequest(c) && s.discoverySseHandler != nil {
			s.discoverySseHandler.ServeHTTP(c.Writer, c.Request)
			return
		}
		eager.ServeHTTP(c.Writer, c.Request)
	}
}
