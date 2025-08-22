package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/api"
	"github.com/mcpjungle/mcpjungle/internal/db"
	"github.com/mcpjungle/mcpjungle/internal/migrations"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/audit"
	"github.com/mcpjungle/mcpjungle/internal/service/config"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp_client"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/spf13/cobra"
)

const (
	BindPortEnvVar  = "PORT"
	BindPortDefault = "8080"

	DBUrlEnvVar = "DATABASE_URL"

	ServerModeEnvVar = "SERVER_MODE"
)

var (
	startServerCmdBindPort    string
	startServerCmdProdEnabled bool
)

var startServerCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the MCP registry server",
	Long: "Starts the MCPJungle HTTP registry server and the MCP Proxy server.\n" +
		"The server is started in Development mode by default, which is ideal for individual users.\n",
	RunE: runStartServer,
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "1",
	},
}

func init() {
	startServerCmd.Flags().StringVar(
		&startServerCmdBindPort,
		"port",
		"",
		fmt.Sprintf("port to bind the server to (overrides env var %s)", BindPortEnvVar),
	)
	startServerCmd.Flags().BoolVar(
		&startServerCmdProdEnabled,
		"prod",
		false,
		fmt.Sprintf(
			"Run the server in Production mode (suitable for enterprises)."+
				" Alternatively, set the %s environment variable ('%s' | '%s')",
			ServerModeEnvVar, model.ModeDev, model.ModeProd,
		),
	)

	rootCmd.AddCommand(startServerCmd)
}

func runStartServer(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load()

	// connect to the DB and run migrations
	dsn := os.Getenv(DBUrlEnvVar)
	dbConn, err := db.NewDBConnection(dsn)
	if err != nil {
		return err
	}
	// Migrations should ideally be decoupled from both the server and the startup phase
	// (should be run as a separate command).
	// However, for the user's convenience, we run them as part of startup command for now.
	if err := migrations.Migrate(dbConn); err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	// determine the port to bind the server to
	port := startServerCmdBindPort
	if port == "" {
		port = os.Getenv(BindPortEnvVar)
	}
	if port == "" {
		port = BindPortDefault
	}

	// create the audit service for tool call logging
	auditConfig := audit.Config{
		Environment: os.Getenv("ENV"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
	}
	if auditConfig.Environment == "" {
		auditConfig.Environment = "development"
	}
	if auditConfig.LogLevel == "" {
		auditConfig.LogLevel = "info"
	}

	auditService, err := audit.NewAuditService(auditConfig)
	if err != nil {
		return fmt.Errorf("failed to create audit service: %v", err)
	}
	defer auditService.Close()

	// create the MCP proxy server
	mcpProxyServer := server.NewMCPServer(
		"MCPJungle Proxy MCP Server",
		"0.0.1",
		server.WithToolCapabilities(true),
	)

	mcpService, err := mcp.NewMCPService(dbConn, mcpProxyServer, auditService)
	if err != nil {
		return fmt.Errorf("failed to create MCP service: %v", err)
	}

	mcpClientService := mcp_client.NewMCPClientService(dbConn)

	configService := config.NewServerConfigService(dbConn)
	userService := user.NewUserService(dbConn)

	// create the API server
	opts := &api.ServerOptions{
		Port:             port,
		MCPProxyServer:   mcpProxyServer,
		MCPService:       mcpService,
		MCPClientService: mcpClientService,
		ConfigService:    configService,
		UserService:      userService,
	}
	s, err := api.NewServer(opts)
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	// determine the server mode
	desiredMode := model.ModeDev
	envMode := os.Getenv(ServerModeEnvVar)
	if envMode != "" {
		// the value of the environment variable is allowed to be case-insensitive
		envMode = strings.ToLower(envMode)

		if envMode != string(model.ModeDev) && envMode != string(model.ModeProd) {
			return fmt.Errorf(
				"invalid value for %s environment variable: '%s', valid values are '%s' and '%s'",
				ServerModeEnvVar, envMode, model.ModeDev, model.ModeProd,
			)
		}

		desiredMode = model.ServerMode(envMode)
	}
	if startServerCmdProdEnabled {
		// If the --prod flag is set, it gets precedence over the environment variable
		desiredMode = model.ModeProd
	}

	// determine server init status
	ok, err := s.IsInitialized()
	if err != nil {
		return fmt.Errorf("failed to check if server is initialized: %v", err)
	}
	if ok {
		// If the server is already initialized, then the mode supplied to this command (desired mode)
		// must match the configured mode.
		mode, err := s.GetMode()
		if err != nil {
			return fmt.Errorf("failed to get server mode: %v", err)
		}
		if desiredMode != mode {
			return fmt.Errorf(
				"server is already initialized in %s mode, cannot start in %s mode",
				mode, desiredMode,
			)
		}
	} else {
		// If server isn't already initialized and the desired mode is dev, silently initialize the server.
		// Individual (dev mode) users need not worry about server initialization.
		if desiredMode == model.ModeDev {
			if err := s.InitDev(); err != nil {
				return fmt.Errorf("failed to initialize server in development mode: %v", err)
			}
		} else {
			// If desired mode is prod, then server initialization is a manual next step to be taken by the user.
			// This is so that they can obtain the admin access token on their client machine.
			fmt.Println(
				"Starting server in Production mode," +
					" don't forget to initialize it by running the `init-server` command",
			)
		}
	}

	fmt.Printf("MCPJungle HTTP server listening on :%s\n\n", port)
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to run the server: %v\n", err)
	}

	return nil
}
