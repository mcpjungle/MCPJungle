package api

import (
	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

func (s *Server) registerInitServerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Mode model.ServerMode `json:"mode" binding:"required,oneof=development enterprise production"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}
		ok, err := s.configService.Init(req.Mode)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to initialize server: " + err.Error()})
			return
		}
		if !ok {
			c.JSON(400, gin.H{"status": "Server already initialized", "mode": req.Mode})
			return
		}
		if req.Mode == model.ModeDev {
			// If the server was successfully initialized and the mode is dev,
			// return a success message without creating an admin user
			c.JSON(200, gin.H{"status": "Server initialized successfully in development mode"})
			return
		}
		// The server was successfully initialized and the mode is enterprise (either ModeEnterprise or ModeProd),
		// create an admin user and return its access token
		admin, err := s.userService.CreateAdminUser()
		if err != nil {
			c.JSON(
				500, gin.H{"error": "Initialization succeeded but failed to create admin user: " + err.Error()},
			)
			return
		}
		payload := gin.H{
			"status":             "Server initialized successfully",
			"admin_access_token": admin.AccessToken,
		}
		c.JSON(200, payload)
	}
}
