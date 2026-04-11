package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestHandleServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "bare ErrNotFound returns 404",
			err:            apierrors.ErrNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   "not found",
		},
		{
			name:           "wrapped ErrNotFound returns 404",
			err:            fmt.Errorf("MCP server xyz not found: %w", apierrors.ErrNotFound),
			expectedStatus: http.StatusNotFound,
			expectedBody:   "not found",
		},
		{
			name:           "double-wrapped ErrNotFound returns 404",
			err:            fmt.Errorf("failed to deregister: %w", fmt.Errorf("MCP server xyz not found: %w", apierrors.ErrNotFound)),
			expectedStatus: http.StatusNotFound,
			expectedBody:   "not found",
		},
		{
			name:           "bare ErrInvalidInput returns 400",
			err:            apierrors.ErrInvalidInput,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid user input",
		},
		{
			name:           "wrapped ErrInvalidInput returns 400",
			err:            fmt.Errorf("invalid access token: %w", apierrors.ErrInvalidInput),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid access token",
		},
		{
			name:           "unrelated error returns 500",
			err:            errors.New("db connection refused"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "db connection refused",
		},
		{
			name:           "wrapped unrelated error returns 500",
			err:            fmt.Errorf("operation failed: %w", errors.New("disk full")),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "disk full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			capturedErr := tt.err
			router.GET("/test", func(c *gin.Context) {
				handleServiceError(c, capturedErr)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			testhelpers.AssertEqual(t, tt.expectedStatus, w.Code)
			testhelpers.AssertStringContains(t, w.Body.String(), tt.expectedBody)
		})
	}
}

// TestErrNotFoundWrapping verifies that errors.Is works correctly through wrapping chains.
func TestErrNotFoundWrapping(t *testing.T) {
	bare := apierrors.ErrNotFound
	wrapped := fmt.Errorf("server xyz not found: %w", apierrors.ErrNotFound)
	doubleWrapped := fmt.Errorf("outer: %w", wrapped)
	unrelated := errors.New("something else")

	testhelpers.AssertTrue(t, errors.Is(bare, apierrors.ErrNotFound), "bare ErrNotFound must match")
	testhelpers.AssertTrue(t, errors.Is(wrapped, apierrors.ErrNotFound), "single-wrapped must match")
	testhelpers.AssertTrue(t, errors.Is(doubleWrapped, apierrors.ErrNotFound), "double-wrapped must match")
	testhelpers.AssertFalse(t, errors.Is(unrelated, apierrors.ErrNotFound), "unrelated error must not match")
}

func TestInvalidInputWrapping(t *testing.T) {
	bare := apierrors.ErrInvalidInput
	wrapped := fmt.Errorf("invalid tool name: %w", apierrors.ErrInvalidInput)
	doubleWrapped := fmt.Errorf("outer: %w", wrapped)
	unrelated := errors.New("something else")

	testhelpers.AssertTrue(t, errors.Is(bare, apierrors.ErrInvalidInput), "bare ErrInvalidInput must match")
	testhelpers.AssertTrue(t, errors.Is(wrapped, apierrors.ErrInvalidInput), "single-wrapped must match")
	testhelpers.AssertTrue(t, errors.Is(doubleWrapped, apierrors.ErrInvalidInput), "double-wrapped must match")
	testhelpers.AssertFalse(t, errors.Is(unrelated, apierrors.ErrInvalidInput), "unrelated error must not match")
}
