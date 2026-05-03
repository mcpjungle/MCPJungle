package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// handleServiceError writes the appropriate HTTP error response for a service-layer error.
// It maps apierrors.ErrNotFound to 404 not found
// and apierrors.ErrInvalidInput to 400 bad request.
// all other errors become 500.
func handleServiceError(c *gin.Context, err error) {
	if errors.Is(err, apierrors.ErrNotFound) {
		c.JSON(http.StatusNotFound, types.APIErrorResponse{Error: err.Error()})
		return
	}
	if errors.Is(err, apierrors.ErrInvalidInput) {
		resp := types.APIErrorResponse{Error: err.Error()}
		if errors.Is(err, apierrors.ErrUpstreamOAuthRequired) {
			resp.Code = apierrors.CodeUpstreamOAuthRequired
		}
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	c.JSON(http.StatusInternalServerError, types.APIErrorResponse{Error: err.Error()})
}
