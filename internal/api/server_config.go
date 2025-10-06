package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
)

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
		admin, err := s.userService.CreateAdminUser()
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
