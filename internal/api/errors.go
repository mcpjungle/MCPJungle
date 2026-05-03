package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
)

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// handleServiceError writes the appropriate HTTP error response for a service-layer error.
// It maps apierrors.ErrNotFound to 404 not found
// and apierrors.ErrInvalidInput to 400 bad request.
// all other errors become 500.
func handleServiceError(c *gin.Context, err error) {
	if errors.Is(err, apierrors.ErrNotFound) {
		c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
		return
	}
	if errors.Is(err, apierrors.ErrInvalidInput) {
		resp := errorResponse{Error: err.Error()}
		if errors.Is(err, apierrors.ErrUpstreamOAuthRequired) {
			resp.Code = apierrors.CodeUpstreamOAuthRequired
		}
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
}
