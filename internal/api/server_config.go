package api

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

// requireInitToken gates POST /init. With no secret configured, only loopback
// callers are allowed. With a secret, callers must supply it in X-Init-Token.
func requireInitToken(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			host, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
			ip := net.ParseIP(host)
			if ip == nil || !ip.IsLoopback() {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Server initialization is restricted to localhost when MCPJUNGLE_INIT_TOKEN is not set",
				})
				return
			}
		} else {
			if c.GetHeader("X-Init-Token") != secret {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Missing or invalid X-Init-Token header",
				})
				return
			}
		}
		c.Next()
	}
}

func (s *Server) registerInitServerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Mode model.ServerMode `json:"mode" binding:"required,oneof=development enterprise production"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}
		ok, err := s.configService.Init(req.Mode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize server: " + err.Error()})
			return
		}
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Server is already initialized"})
			return
		}
		if req.Mode == model.ModeDev {
			// If the server was successfully initialized and the mode is dev,
			// return a success message without creating an admin user
			c.JSON(http.StatusOK, gin.H{"status": "Server initialized successfully in development mode"})
			return
		}
		// The server was successfully initialized and the mode is enterprise (either ModeEnterprise or ModeProd),
		// create an admin user and return its access token
		admin, err := s.userService.CreateAdminUser("")
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "Initialization succeeded but failed to create admin user: " + err.Error()},
			)
			return
		}
		payload := gin.H{
			"status":             "Server initialized successfully",
			"admin_access_token": admin.AccessToken,
		}
		c.JSON(http.StatusOK, payload)
	}
}
