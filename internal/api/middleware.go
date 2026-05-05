package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// requireInitialized is middleware to reject requests to certain routes if the server is not initialized
func (s *Server) requireInitialized() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := s.configService.GetConfig()
		if err != nil || !cfg.Initialized {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "server is not initialized"})
			return
		}
		// propagate the server mode in context for other middleware/handlers to use
		c.Set("mode", cfg.Mode)
		c.Next()
	}
}

// verifyUserAuthForAPIAccess is middleware that checks for a valid user token if the server is in enterprise mode.
// this middleware doesn't care about the role of the user, it just verifies that they're authenticated.
func (s *Server) verifyUserAuthForAPIAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode, exists := c.Get("mode")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server mode not found in context"})
			return
		}
		m, ok := mode.(model.ServerMode)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid server mode in context"})
			return
		}
		if m == model.ModeDev {
			// no auth is required in case of dev mode
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing access token"})
			return
		}

		// Verify that the token is valid and corresponds to a user
		authenticatedUser, err := s.userService.GetUserByAccessToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid access token: " + err.Error()})
			return
		}

		// Store user in context for potential role checks in subsequent handlers
		c.Set("user", authenticatedUser)
		c.Next()
	}
}

// requireAdminUser is middleware that ensures the authenticated user has an admin role when in enterprise mode.
// It assumes that verifyUserAuthForAPIAccess middleware has already run and set the user in context.
func (s *Server) requireAdminUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode, exists := c.Get("mode")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server mode not found in context"})
			return
		}
		m, ok := mode.(model.ServerMode)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid server mode in context"})
			return
		}
		if m == model.ModeDev {
			// no admin check is required in dev mode
			c.Next()
			return
		}

		authenticatedUser, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user is not authenticated"})
			return
		}

		u, ok := authenticatedUser.(*model.User)
		if ok && u.Role == types.UserRoleAdmin {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user is not authorized to perform this action"})
	}
}

// requireServerMode is middleware that checks if the server is in a specific mode.
// If not, the request is rejected with a 403 Forbidden status.
// This is useful for routes that should only be accessible in certain modes (e.g., enterprise-only features).
// NOTE: ModeProd is supported for backwards compatibility, it is equivalent to ModeEnterprise.
func (s *Server) requireServerMode(m model.ServerMode) gin.HandlerFunc {
	return func(c *gin.Context) {
		mode, exists := c.Get("mode")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server mode not found in context"})
			return
		}
		currentMode, ok := mode.(model.ServerMode)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid server mode in context"})
			return
		}

		if currentMode == m {
			// current mode matches the required mode, allow access
			c.Next()
			return
		}
		if model.IsEnterpriseMode(currentMode) && model.IsEnterpriseMode(m) {
			// both current and required modes are enterprise modes, allow access
			c.Next()
			return
		}
		// current mode does not match the required mode, reject the request
		c.AbortWithStatusJSON(
			http.StatusForbidden,
			gin.H{"error": fmt.Sprintf("this request is only allowed in %s mode", m)},
		)
	}
}

// blockBoundClientsFromGlobalMCP is middleware for the global /mcp endpoint.
// When RequireClientTGBinding (strict mode) is enabled, it rejects clients that have a BoundToolGroup
// set — those clients must use the group-specific endpoint, not the global one.
// This middleware must run after checkAuthForMcpProxyAccess.
func (s *Server) blockBoundClientsFromGlobalMCP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.requireClientTGBinding {
			c.Next()
			return
		}

		raw := c.Request.Context().Value("client")
		if raw == nil {
			// dev mode or unauthenticated — let through (other middleware handles auth)
			c.Next()
			return
		}
		client, ok := raw.(*model.McpClient)
		if !ok || client == nil {
			c.Next()
			return
		}

		if client.BoundToolGroup != nil {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				gin.H{
					"error": fmt.Sprintf(
						"client %q is bound to tool group %q and must use the group-specific endpoint /v0/groups/%s/mcp",
						client.Name, *client.BoundToolGroup, *client.BoundToolGroup,
					),
				},
			)
			return
		}
		c.Next()
	}
}

// enforceToolGroupBinding is middleware that restricts a bound MCP client to its designated Tool Group.
// It must run after checkAuthForMcpProxyAccess (which injects the client into the request context).
// If the client has a BoundToolGroup set, the URL parameter :name must match it exactly.
// Clients without a binding (BoundToolGroup == nil) may access any group endpoint (legacy behaviour).
// In dev mode the middleware is a no-op (no client is injected into context).
func (s *Server) enforceToolGroupBinding() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the client injected by checkAuthForMcpProxyAccess.
		// In dev mode the client is not injected, so we skip enforcement.
		raw := c.Request.Context().Value("client")
		if raw == nil {
			c.Next()
			return
		}
		client, ok := raw.(*model.McpClient)
		if !ok || client == nil {
			c.Next()
			return
		}

		if client.BoundToolGroup == nil {
			// Unbound client — legacy behaviour, allow access to any group.
			c.Next()
			return
		}

		groupName := c.Param("name")
		if *client.BoundToolGroup != groupName {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				gin.H{
					"error": fmt.Sprintf(
						"client %q is bound to tool group %q and may not access group %q",
						client.Name, *client.BoundToolGroup, groupName,
					),
				},
			)
			return
		}
		c.Next()
	}
}

// checkAuthForMcpProxyAccess is middleware for MCP proxy that checks for a valid MCP client token
// if the server is in enterprise mode.
// In development mode, mcp clients do not require auth to access the MCP proxy.
func (s *Server) checkAuthForMcpProxyAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		mode, exists := c.Get("mode")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server mode not found in context"})
			return
		}
		m, ok := mode.(model.ServerMode)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid server mode in context"})
			return
		}

		// the gin context doesn't get passed down to the MCP proxy server, so we need to
		// set values in the underlying request's context to be able to access them from proxy.
		ctx := context.WithValue(c.Request.Context(), "mode", m)
		c.Request = c.Request.WithContext(ctx)

		if m == model.ModeDev {
			// no auth is required in case of dev mode
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing MCP client access token"})
			return
		}
		client, err := s.mcpClientService.GetClientByToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid MCP client token"})
			return
		}

		// inject the authenticated MCP client in context for the proxy to use
		ctx = context.WithValue(c.Request.Context(), "client", client)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
