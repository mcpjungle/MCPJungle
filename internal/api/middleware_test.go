package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/config"
	"github.com/mcpjungle/mcpjungle/internal/service/mcpclient"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/gorm"
)

func TestRequireInitialized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupConfig    func(*gorm.DB) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "server is initialized",
			setupConfig: func(testDB *gorm.DB) error {
				configService := config.NewServerConfigService(testDB)
				_, err := configService.Init(model.ModeDev)
				return err
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name: "server is not initialized",
			setupConfig: func(testDB *gorm.DB) error {
				cfg := model.ServerConfig{
					Initialized: false,
					Mode:        model.ModeDev,
				}
				return testDB.Create(&cfg).Error
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"server is not initialized"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := testhelpers.SetupTestDB(t)
			defer setup.Cleanup()
			testDB := setup.DB
			configService := config.NewServerConfigService(testDB)

			err := tt.setupConfig(testDB)
			if err != nil {
				t.Fatalf("Setup config failed: %v", err)
			}

			server := &Server{configService: configService}
			router := gin.New()
			router.Use(server.requireInitialized())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestVerifyUserAuthForAPIAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()
	testDB := setup.DB

	userService := user.NewUserService(testDB)

	tests := []struct {
		name           string
		mode           model.ServerMode
		authHeader     string
		setupUser      func() error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "dev mode - no auth required",
			mode:           model.ModeDev,
			authHeader:     "",
			setupUser:      func() error { return nil },
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:       "enterprise mode - valid token",
			mode:       model.ModeEnterprise,
			authHeader: "Bearer test-token",
			setupUser: func() error {
				_, err := userService.CreateAdminUser()
				if err != nil {
					return err
				}
				var u model.User
				err = testDB.Where("username = ?", "admin").First(&u).Error
				if err != nil {
					return err
				}
				u.AccessToken = "test-token"
				return testDB.Save(&u).Error
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "enterprise mode - missing token",
			mode:           model.ModeEnterprise,
			authHeader:     "",
			setupUser:      func() error { return nil },
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"missing access token"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupUser()
			if err != nil {
				t.Fatalf("Setup user failed: %v", err)
			}

			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tt.mode != "" {
					c.Set("mode", tt.mode)
				}
			})
			server := &Server{userService: userService}
			router.Use(server.verifyUserAuthForAPIAccess())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestRequireAdminUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testDB := testhelpers.SetupTestDB(t).DB
	userService := user.NewUserService(testDB)

	tests := []struct {
		name           string
		mode           model.ServerMode
		user           any
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "dev mode - no admin check required",
			mode:           model.ModeDev,
			user:           nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name: "enterprise mode - admin user",
			mode: model.ModeEnterprise,
			user: &model.User{
				Model:    gorm.Model{ID: 1},
				Username: "admin",
				Role:     types.UserRoleAdmin,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name: "enterprise mode - regular user",
			mode: model.ModeEnterprise,
			user: &model.User{
				Model:    gorm.Model{ID: 1},
				Username: "user",
				Role:     types.UserRoleUser,
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"user is not authorized to perform this action"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tt.mode != "" {
					c.Set("mode", tt.mode)
				}
				if tt.user != nil {
					c.Set("user", tt.user)
				}
			})
			server := &Server{userService: userService}
			router.Use(server.requireAdminUser())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestRequireServerMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		contextMode    model.ServerMode
		requiredMode   model.ServerMode
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "matching mode - dev",
			contextMode:    model.ModeDev,
			requiredMode:   model.ModeDev,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "non-matching mode - dev required, enterprise context",
			contextMode:    model.ModeEnterprise,
			requiredMode:   model.ModeDev,
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"this request is only allowed in development mode"}`,
		},
		{
			name:           "non-matching mode - dev required, prod context",
			contextMode:    model.ModeProd,
			requiredMode:   model.ModeDev,
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"this request is only allowed in development mode"}`,
		},
		{
			name:           "enterprise required, prod context (deprecated)",
			contextMode:    model.ModeProd,
			requiredMode:   model.ModeEnterprise,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "prod required, enterprise context (deprecated)",
			contextMode:    model.ModeEnterprise,
			requiredMode:   model.ModeProd,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "prod required, prod context (deprecated)",
			contextMode:    model.ModeProd,
			requiredMode:   model.ModeProd,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "enterprise required, enterprise context",
			contextMode:    model.ModeEnterprise,
			requiredMode:   model.ModeEnterprise,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tt.contextMode != "" {
					c.Set("mode", tt.contextMode)
				}
			})
			server := &Server{}
			router.Use(server.requireServerMode(tt.requiredMode))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestCheckAuthForMcpProxyAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()
	testDB := setup.DB

	mcpClientService := mcpclient.NewMCPClientService(testDB)

	tests := []struct {
		name           string
		mode           model.ServerMode
		authHeader     string
		setupClient    func() error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "dev mode - no auth required",
			mode:           model.ModeDev,
			authHeader:     "",
			setupClient:    func() error { return nil },
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:       "enterprise mode - valid token",
			mode:       model.ModeEnterprise,
			authHeader: "Bearer test-token",
			setupClient: func() error {
				client := model.McpClient{
					Name:        "test-client",
					Description: "Test client",
					AllowList:   []byte("[]"),
				}
				_, err := mcpClientService.CreateClient(client)
				if err != nil {
					return err
				}
				var c model.McpClient
				err = testDB.Where("name = ?", "test-client").First(&c).Error
				if err != nil {
					return err
				}
				c.AccessToken = "test-token"
				return testDB.Save(&c).Error
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "enterprise mode - missing token",
			mode:           model.ModeEnterprise,
			authHeader:     "",
			setupClient:    func() error { return nil },
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"missing MCP client access token"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupClient()
			if err != nil {
				t.Fatalf("Setup client failed: %v", err)
			}

			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tt.mode != "" {
					c.Set("mode", tt.mode)
				}
			})
			server := &Server{mcpClientService: mcpClientService}
			router.Use(server.checkAuthForMcpProxyAccess())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestMiddlewareIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()
	testDB := setup.DB

	configService := config.NewServerConfigService(testDB)
	userService := user.NewUserService(testDB)

	// Setup config
	_, err := configService.Init(model.ModeEnterprise)
	if err != nil {
		t.Fatalf("Setup config failed: %v", err)
	}

	// Setup user
	_, err = userService.CreateAdminUser()
	if err != nil {
		t.Fatalf("Setup user failed: %v", err)
	}

	// Update user with known token and admin role
	var u model.User
	err = testDB.Where("username = ?", "admin").First(&u).Error
	if err != nil {
		t.Fatalf("Failed to find admin user: %v", err)
	}
	u.AccessToken = "valid-token"
	u.Role = types.UserRoleAdmin
	err = testDB.Save(&u).Error
	if err != nil {
		t.Fatalf("Failed to save admin user: %v", err)
	}

	server := &Server{
		configService: configService,
		userService:   userService,
	}
	router := gin.New()
	router.Use(server.requireInitialized())
	router.Use(server.verifyUserAuthForAPIAccess())
	router.Use(server.requireAdminUser())
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "admin access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	expectedBody := `{"status":"admin access granted"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

// injectClientIntoRequest returns a copy of req with the McpClient injected into its context,
// simulating what checkAuthForMcpProxyAccess does in the real request path.
func injectClientIntoRequest(req *http.Request, client *model.McpClient) *http.Request {
	ctx := context.WithValue(req.Context(), "client", client)
	return req.WithContext(ctx)
}

// strPtr is a convenience helper for *string literals in tests.
func strPtr(s string) *string { return &s }

// TestEnforceToolGroupBinding_BoundClientMatchingGroup verifies that a client bound to "grp-a"
// can reach /groups/grp-a/* without being blocked.
func TestEnforceToolGroupBinding_BoundClientMatchingGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{}

	router := gin.New()
	router.GET("/groups/:name/mcp", s.enforceToolGroupBinding(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	client := &model.McpClient{Name: "c1", BoundToolGroup: strPtr("grp-a")}
	req := injectClientIntoRequest(httptest.NewRequest(http.MethodGet, "/groups/grp-a/mcp", nil), client)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusOK, w.Code)
}

// TestEnforceToolGroupBinding_BoundClientWrongGroup verifies that a client bound to "grp-a"
// gets 403 when it tries to access /groups/grp-b/*.
func TestEnforceToolGroupBinding_BoundClientWrongGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{}

	router := gin.New()
	router.GET("/groups/:name/mcp", s.enforceToolGroupBinding(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	client := &model.McpClient{Name: "c1", BoundToolGroup: strPtr("grp-a")}
	req := injectClientIntoRequest(httptest.NewRequest(http.MethodGet, "/groups/grp-b/mcp", nil), client)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusForbidden, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "grp-a")
}

// TestEnforceToolGroupBinding_UnboundClientAnyGroup verifies that a client with no BoundToolGroup
// (legacy/unbound) can reach any group endpoint — backwards-compat path.
func TestEnforceToolGroupBinding_UnboundClientAnyGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{}

	router := gin.New()
	router.GET("/groups/:name/mcp", s.enforceToolGroupBinding(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	client := &model.McpClient{Name: "legacy", BoundToolGroup: nil}
	req := injectClientIntoRequest(httptest.NewRequest(http.MethodGet, "/groups/any-group/mcp", nil), client)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusOK, w.Code)
}

// TestEnforceToolGroupBinding_DevModeNoClient verifies that the middleware is a no-op when no
// client is in the context (dev mode — checkAuthForMcpProxyAccess skips injection).
func TestEnforceToolGroupBinding_DevModeNoClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{}

	router := gin.New()
	router.GET("/groups/:name/mcp", s.enforceToolGroupBinding(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/groups/any-group/mcp", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusOK, w.Code)
}

// TestBlockBoundClientsFromGlobalMCP_StrictMode_BoundClientBlocked verifies that in strict mode
// a client with BoundToolGroup is rejected from the global /mcp endpoint.
func TestBlockBoundClientsFromGlobalMCP_StrictMode_BoundClientBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{requireClientTGBinding: true}

	router := gin.New()
	router.POST("/mcp", s.blockBoundClientsFromGlobalMCP(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	client := &model.McpClient{Name: "c1", BoundToolGroup: strPtr("code")}
	req := injectClientIntoRequest(httptest.NewRequest(http.MethodPost, "/mcp", nil), client)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusForbidden, w.Code)
	testhelpers.AssertStringContains(t, w.Body.String(), "code")
}

// TestBlockBoundClientsFromGlobalMCP_StrictMode_UnboundClientAllowed verifies that in strict mode
// a client without BoundToolGroup can still hit the global /mcp endpoint.
func TestBlockBoundClientsFromGlobalMCP_StrictMode_UnboundClientAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{requireClientTGBinding: true}

	router := gin.New()
	router.POST("/mcp", s.blockBoundClientsFromGlobalMCP(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	client := &model.McpClient{Name: "legacy", BoundToolGroup: nil}
	req := injectClientIntoRequest(httptest.NewRequest(http.MethodPost, "/mcp", nil), client)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusOK, w.Code)
}

// TestBlockBoundClientsFromGlobalMCP_NonStrictMode_BoundClientAllowed verifies that when strict
// mode is OFF, even a bound client can reach the global /mcp endpoint.
func TestBlockBoundClientsFromGlobalMCP_NonStrictMode_BoundClientAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{requireClientTGBinding: false}

	router := gin.New()
	router.POST("/mcp", s.blockBoundClientsFromGlobalMCP(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	client := &model.McpClient{Name: "c1", BoundToolGroup: strPtr("code")}
	req := injectClientIntoRequest(httptest.NewRequest(http.MethodPost, "/mcp", nil), client)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	testhelpers.AssertEqual(t, http.StatusOK, w.Code)
}
