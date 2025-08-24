package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/config"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp_client"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
)

const V0PathPrefix = "/api/v0"

type ServerOptions struct {
	// Port is the HTTP ports to bind the server to
	Port string

	MCPProxyServer   *server.MCPServer
	MCPService       *mcp.MCPService
	MCPClientService *mcp_client.McpClientService
	ConfigService    *config.ServerConfigService
	UserService      *user.UserService
}

// Server represents the MCPJungle registry server that handles MCP proxy and API requests
type Server struct {
	port   string
	router *gin.Engine

	mcpProxyServer   *server.MCPServer
	mcpService       *mcp.MCPService
	mcpClientService *mcp_client.McpClientService

	configService *config.ServerConfigService
	userService   *user.UserService
}

// NewServer initializes a new Gin server for MCPJungle registry and MCP proxy
func NewServer(opts *ServerOptions) (*Server, error) {
	r, err := newRouter(opts)
	if err != nil {
		return nil, err
	}
	s := &Server{
		port:             opts.Port,
		router:           r,
		mcpProxyServer:   opts.MCPProxyServer,
		mcpService:       opts.MCPService,
		mcpClientService: opts.MCPClientService,
		configService:    opts.ConfigService,
		userService:      opts.UserService,
	}
	return s, nil
}

// IsInitialized returns true if the server is initialized
func (s *Server) IsInitialized() (bool, error) {
	c, err := s.configService.GetConfig()
	if err != nil {
		return false, fmt.Errorf("failed to get server config: %w", err)
	}
	return c.Initialized, nil
}

// GetMode returns the server mode if the server is initialized, otherwise an error
func (s *Server) GetMode() (model.ServerMode, error) {
	ok, err := s.IsInitialized()
	if err != nil {
		return "", fmt.Errorf("failed to check if server is initialized: %w", err)
	}
	if !ok {
		return "", fmt.Errorf("server is not initialized")
	}
	c, err := s.configService.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get server config: %w", err)
	}
	return c.Mode, nil
}

// InitDev initializes the server configuration in the Development mode.
// This method does not create an admin user because that is irrelevant in dev mode.
func (s *Server) InitDev() error {
	_, err := s.configService.Init(model.ModeDev)
	if err != nil {
		return fmt.Errorf("failed to initialize server config in dev mode: %w", err)
	}
	return nil
}

// Start runs the Gin server (blocking call)
func (s *Server) Start() error {
	if err := s.router.Run(":" + s.port); err != nil {
		return fmt.Errorf("failed to run the server: %w", err)
	}
	return nil
}

// newRouter sets up the Gin router with the MCP proxy server and API endpoints.
func newRouter(opts *ServerOptions) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET(
		"/health",
		func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		},
	)

	r.POST("/init", registerInitServerHandler(opts.ConfigService, opts.UserService))

	requireProdMode := requireServerMode(model.ModeProd)

	// Set up the MCP proxy server on /mcp
	streamableHttpServer := server.NewStreamableHTTPServer(opts.MCPProxyServer)
	r.Any(
		"/mcp",
		requireInitialized(opts.ConfigService),
		checkAuthForMcpProxyAccess(opts.MCPClientService),
		gin.WrapH(streamableHttpServer),
	)

	// Setup /v0 API endpoints
	apiV0 := r.Group(
		V0PathPrefix,
		requireInitialized(opts.ConfigService),
		verifyUserAuthForAPIAccess(opts.UserService),
	)

	// endpoints accessible by a standard user in production mode or anyone in development mode
	userAPI := apiV0.Group("/")
	{
		userAPI.GET("/servers", listServersHandler(opts.MCPService))

		userAPI.GET("/tools", listToolsHandler(opts.MCPService))
		userAPI.POST("/tools/invoke", invokeToolHandler(opts.MCPService))
		userAPI.GET("/tool", getToolHandler(opts.MCPService))

		userAPI.GET("/users/whoami", requireProdMode, whoAmIHandler())
	}

	// endpoints only accessible by an admin user in production mode or anyone in development mode
	adminAPI := apiV0.Group("/", requireAdminUser())
	{
		adminAPI.POST("/servers", registerServerHandler(opts.MCPService))
		adminAPI.DELETE("/servers/:name", deregisterServerHandler(opts.MCPService))

		adminAPI.POST("/tools/enable", enableToolsHandler(opts.MCPService))
		adminAPI.POST("/tools/disable", disableToolsHandler(opts.MCPService))

		// endpoints for managing MCP clients (production mode only)
		adminAPI.GET(
			"/clients",
			requireProdMode,
			listMcpClientsHandler(opts.MCPClientService),
		)
		adminAPI.POST(
			"/clients",
			requireProdMode,
			createMcpClientHandler(opts.MCPClientService),
		)
		adminAPI.DELETE(
			"/clients/:name",
			requireProdMode,
			deleteMcpClientHandler(opts.MCPClientService),
		)

		// endpoints for managing human users (production mode only)
		adminAPI.POST("/users",
			requireProdMode,
			createUserHandler(opts.UserService),
		)
		adminAPI.GET("/users",
			requireProdMode,
			listUsersHandler(opts.UserService),
		)
		adminAPI.DELETE("/users/:username",
			requireProdMode,
			deleteUserHandler(opts.UserService),
		)
	}

	return r, nil
}
