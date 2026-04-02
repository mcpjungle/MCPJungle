package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
)

// handleServiceError writes the appropriate HTTP error response for a service-layer error.
// It maps apierrors.ErrNotFound to 404; all other errors become 500.
func handleServiceError(c *gin.Context, err error) {
	if errors.Is(err, apierrors.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
