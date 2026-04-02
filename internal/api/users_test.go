package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestUpdateUserHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	s := &Server{userService: user.NewUserService(setup.DB)}
	router := gin.New()
	router.PUT("/users/:username", s.updateUserHandler())

	req := httptest.NewRequest(http.MethodPut, "/users/ghost",
		strings.NewReader(`{"access_token":"validtoken123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusNotFound, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "not found")
}

func TestDeleteUserHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	s := &Server{userService: user.NewUserService(setup.DB)}
	router := gin.New()
	router.DELETE("/users/:username", s.deleteUserHandler())

	req := httptest.NewRequest(http.MethodDelete, "/users/ghost", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusNotFound, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "not found")
}

func TestDeleteUserHandler_Exists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	setup.CreateTestUser("regularuser", types.UserRoleUser, "sometoken123")

	s := &Server{userService: user.NewUserService(setup.DB)}
	router := gin.New()
	router.DELETE("/users/:username", s.deleteUserHandler())

	req := httptest.NewRequest(http.MethodDelete, "/users/regularuser", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusNoContent, w.Code)
}
