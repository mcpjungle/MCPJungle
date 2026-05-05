package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
)

// handleServiceError writes the appropriate HTTP error response for a service-layer error.
// It maps apierrors.ErrNotFound to 404 not found,
// apierrors.ErrInvalidInput to 400 bad request,
// and apierrors.ErrConflict to 409 conflict.
// all other errors become 500.
func handleServiceError(c *gin.Context, err error) {
	if errors.Is(err, apierrors.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, apierrors.ErrInvalidInput) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, apierrors.ErrConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
