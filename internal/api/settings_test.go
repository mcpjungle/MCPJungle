package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/config"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestGetSettingsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	err := setup.DB.Create(&model.ServerConfig{
		Mode:        model.ModeDev,
		Initialized: true,
	}).Error
	testhelpers.AssertNoError(t, err)

	s := &Server{configService: config.NewServerConfigService(setup.DB)}
	router := gin.New()
	router.GET("/settings", s.getSettingsHandler())

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusOK, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), `"initialized":true`)
	testhelpers.AssertStringContains(t, w.Body.String(), `"mode":"development"`)
	testhelpers.AssertStringContains(t, w.Body.String(), `"version":`)
}
