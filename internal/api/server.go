// Package api provides HTTP API functionality for the MCPJungle server.
package api

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/config"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/mcpclient"
	"github.com/mcpjungle/mcpjungle/internal/service/toolgroup"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/mcpjungle/mcpjungle/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

const (
	V0PathPrefix    = "/v0"
	V0ApiPathPrefix = "/api" + V0PathPrefix
)

type ServerOptions struct {
	// MCPProxyServer is the MCP proxy server instance that contains tools for all MCP servers
	// using the stdio or streamable http transport.
	MCPProxyServer *server.MCPServer
	// SseMcpProxyServer is the MCP proxy server instance that contains tools for all MCP servers
	// using the SSE transport.
	// sse tools are kept separate because SSE is supported for backward compatibility reasons, and
	// we don't want it to interfere with the usual mcp proxy server.
	// Both sse & streamable http use http, and we don't want to mix them up either.
	SseMcpProxyServer *server.MCPServer

	MCPService       *mcp.MCPService
	MCPClientService *mcpclient.McpClientService
	ConfigService    *config.ServerConfigService
	UserService      *user.UserService
	ToolGroupService *toolgroup.ToolGroupService

	OtelProviders *telemetry.Providers
	Metrics       telemetry.CustomMetrics
}

// Server represents the MCPJungle registry server that handles MCP proxy and API requests
type Server struct {
	router *gin.Engine

	mcpProxyServer    *server.MCPServer
	sseMcpProxyServer *server.MCPServer

	mcpService       *mcp.MCPService
	mcpClientService *mcpclient.McpClientService

	configService    *config.ServerConfigService
	userService      *user.UserService
	toolGroupService *toolgroup.ToolGroupService

	otelProviders *telemetry.Providers
	metrics       telemetry.CustomMetrics

	// groupMcpServers keeps track of mcp-go's server.SSEServer instances created for each tool group.
	// These instances serve the requests made to tool groups' SSE tools.
	// We need to maintain one instance for each group for sse to work correctly.
	groupSseServers sync.Map
}

// NewServer initializes a new Gin server for MCPJungle registry and MCP proxy
func NewServer(opts *ServerOptions) (*Server, error) {
	s := &Server{
		mcpProxyServer:    opts.MCPProxyServer,
		sseMcpProxyServer: opts.SseMcpProxyServer,
		mcpService:        opts.MCPService,
		mcpClientService:  opts.MCPClientService,
		configService:     opts.ConfigService,
		userService:       opts.UserService,
		toolGroupService:  opts.ToolGroupService,
		otelProviders:     opts.OtelProviders,
		metrics:           opts.Metrics,
	}

	// Set up the router after the server is fully initialized
	r, err := s.setupRouter()
	if err != nil {
		return nil, err
	}
	s.router = r

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

// Router returns the underlying HTTP handler for use with a custom HTTP server.
// This is useful for graceful shutdown support.
func (s *Server) Router() http.Handler {
	return s.router
}

// setupRouter sets up the Gin router with the MCP proxy server and API endpoints.
func (s *Server) setupRouter() (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// if otel is enabled, setup prometheus metrics endpoint
	if s.otelProviders != nil && s.otelProviders.IsEnabled() {
		// instrument gin
		r.Use(otelgin.Middleware(s.otelProviders.ServiceName()))

		// expose prometheus metrics endpoint
		r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	r.GET(
		"/health",
		func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		},
	)

	r.GET(
		"/metadata",
		func(c *gin.Context) {
			m := &types.ServerMetadata{
				Version: version.GetVersion(),
			}
			c.JSON(http.StatusOK, m)
		},
	)

	r.POST("/init", s.registerInitServerHandler())

	requireEnterpriseMode := s.requireServerMode(model.ModeEnterprise)

	// Set up the MCP proxy server on /mcp
	streamableHTTPServer := server.NewStreamableHTTPServer(s.mcpProxyServer)
	r.Any(
		"/mcp",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		gin.WrapH(streamableHTTPServer),
	)

	r.Any(
		V0PathPrefix+"/groups/:name/mcp",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		s.toolGroupMCPServerCallHandler(),
	)

	// Set up the SSE transport-based MCP proxy server for the global /sse endpoint
	sseServer := server.NewSSEServer(s.sseMcpProxyServer)
	r.Any(
		"/sse",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		gin.WrapH(sseServer.SSEHandler()),
	)
	r.Any(
		"/message",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		gin.WrapH(sseServer.MessageHandler()),
	)

	r.Any(
		V0PathPrefix+"/groups/:name/sse",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		s.toolGroupSseMCPServerCallHandler(),
	)
	r.Any(
		V0PathPrefix+"/groups/:name/message",
		s.requireInitialized(),
		s.checkAuthForMcpProxyAccess(),
		s.toolGroupSseMCPServerCallMessageHandler(),
	)

	// Setup /v0 API endpoints
	apiV0 := r.Group(
		V0ApiPathPrefix,
		s.requireInitialized(),
		s.verifyUserAuthForAPIAccess(),
	)

	// endpoints accessible by a standard user in enterprise mode or anyone in development mode
	userAPI := apiV0.Group("/")
	{
		userAPI.GET("/servers", s.listServersHandler())

		userAPI.GET("/tools", s.listToolsHandler())
		userAPI.POST("/tools/invoke", s.invokeToolHandler())
		userAPI.GET("/tool", s.getToolHandler())

		// Prompt endpoints
		userAPI.GET("/prompts", s.listPromptsHandler())
		userAPI.GET("/prompt", s.getPromptHandler())
		userAPI.POST("/prompts/render", s.getPromptWithArgsHandler())

		userAPI.GET("/users/whoami", requireEnterpriseMode, s.whoAmIHandler())
	}

	// endpoints only accessible by an admin user in enterprise mode or anyone in development mode
	adminAPI := apiV0.Group("/", s.requireAdminUser())
	{
		adminAPI.POST("/servers", s.registerServerHandler())
		adminAPI.DELETE("/servers/:name", s.deregisterServerHandler())
		adminAPI.POST("/servers/:name/enable", s.enableServerHandler())
		adminAPI.POST("/servers/:name/disable", s.disableServerHandler())

		// this endpoint is restricted to admins only because it can potentially expose sensitive information
		// like bearer tokens.
		adminAPI.GET("/server_configs", s.getServerConfigsHandler())

		adminAPI.POST("/tools/enable", s.enableToolsHandler())
		adminAPI.POST("/tools/disable", s.disableToolsHandler())

		adminAPI.POST("/prompts/enable", s.enablePromptsHandler())
		adminAPI.POST("/prompts/disable", s.disablePromptsHandler())

		// endpoints for managing MCP clients (enterprise mode only)
		adminAPI.GET(
			"/clients",
			requireEnterpriseMode,
			s.listMcpClientsHandler(),
		)
		adminAPI.POST(
			"/clients",
			requireEnterpriseMode,
			s.createMcpClientHandler(),
		)
		adminAPI.PUT(
			"/clients/:name",
			requireEnterpriseMode,
			s.updateMcpClientHandler(),
		)
		adminAPI.DELETE(
			"/clients/:name",
			requireEnterpriseMode,
			s.deleteMcpClientHandler(),
		)

		// endpoints for managing human users (enterprise mode only)
		adminAPI.POST(
			"/users",
			requireEnterpriseMode,
			s.createUserHandler(),
		)
		adminAPI.GET(
			"/users",
			requireEnterpriseMode,
			s.listUsersHandler(),
		)
		adminAPI.DELETE(
			"/users/:username",
			requireEnterpriseMode,
			s.deleteUserHandler(),
		)
		adminAPI.PUT(
			"/users/:username",
			requireEnterpriseMode,
			s.updateUserHandler(),
		)

		// endpoints for managing tool groups
		adminAPI.POST("/tool-groups", s.createToolGroupHandler())
		adminAPI.GET("/tool-groups/:name", s.getToolGroupHandler())
		adminAPI.GET("/tool-groups/:name/effective-tools", s.getToolGroupEffectiveToolsHandler())
		adminAPI.GET("/tool-groups", s.listToolGroupsHandler())
		adminAPI.DELETE("/tool-groups/:name", s.deleteToolGroupHandler())
		adminAPI.PUT("/tool-groups/:name", s.updateToolGroupHandler())
	}

	return r, nil
}
