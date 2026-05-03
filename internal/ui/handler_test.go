package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestHandlerServesUnavailableWhenAssetsMissing(t *testing.T) {
	handler, err := NewHandlerFromEnv()
	testhelpers.AssertNoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/ui/dashboard", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		testhelpers.AssertStringContains(t, w.Body.String(), "<!doctype html>")
		return
	}

	testhelpers.AssertEqual(t, http.StatusServiceUnavailable, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "UI assets not built")
}
