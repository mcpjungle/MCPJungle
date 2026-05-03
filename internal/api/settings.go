package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/pkg/version"
)

func (s *Server) getSettingsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := s.configService.GetConfig()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get server config: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"initialized": cfg.Initialized,
			"mode":        cfg.Mode,
			"version":     version.GetVersion(),
		})
	}
}
