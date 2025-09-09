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
	"github.com/mcpjungle/mcpjungle/internal/service/config"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/mcpclient"
	"github.com/mcpjungle/mcpjungle/internal/service/toolgroup"
	"github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/spf13/cobra"
)

const (
	BindPortEnvVar  = "PORT"
	BindPortDefault = "8080"

	DBUrlEnvVar            = "DATABASE_URL"
	ServerModeEnvVar       = "SERVER_MODE"
	TelemetryEnabledEnvVar = "OTEL_ENABLED"
)

var (
	startServerCmdBindPort    string
	startServerCmdProdEnabled bool
)

var startServerCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the MCPJungle server",
	Long: "Starts the MCPJungle HTTP registry server and the MCP Proxy server.\n" +
		"The server is started in development mode by default, which is ideal for individual users.\n" +
		"Teams & Enterprises should run mcpjungle in production mode.\n",
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
		fmt.Sprintf("port to bind the HTTP server to (overrides env var %s)", BindPortEnvVar),
	)
	startServerCmd.Flags().BoolVar(
		&startServerCmdProdEnabled,
		"prod",
		false,
		fmt.Sprintf(
			"Run the server in Production mode (ideal for teams and enterprises)."+
				" Alternatively, set the %s environment variable ('%s' | '%s')",
			ServerModeEnvVar, model.ModeDev, model.ModeProd,
		),
	)

	rootCmd.AddCommand(startServerCmd)
}

// getDesiredServerMode returns the desired server mode for mcpjungle server.
// unless explicitly specified, the desired mode is dev
func getDesiredServerMode() (model.ServerMode, error) {
	desiredServerMode := model.ModeDev

	envMode := os.Getenv(ServerModeEnvVar)
	if envMode != "" {
		// the value of the environment variable is allowed to be case-insensitive
		envMode = strings.ToLower(envMode)

		if envMode != string(model.ModeDev) && envMode != string(model.ModeProd) {
			return "", fmt.Errorf(
				"invalid value for %s environment variable: '%s', valid values are '%s' and '%s'",
				ServerModeEnvVar, envMode, model.ModeDev, model.ModeProd,
			)
		}

		desiredServerMode = model.ServerMode(envMode)
	}
	if startServerCmdProdEnabled {
		// If the --prod flag is set, it gets precedence over the environment variable
		desiredServerMode = model.ModeProd
	}

	return desiredServerMode, nil
}

// isTelemetryEnabled returns true if telemetry should be enabled.
// If an env var is specified, it takes precedence over the defaults.
// Otherwise, by default, telemetry is disabled in dev mode and enabled in prod mode.
func isTelemetryEnabled(desiredServerMode model.ServerMode) (bool, error) {
	telemetryEnabled := desiredServerMode == model.ModeProd

	envTelemetryEnabled := os.Getenv(TelemetryEnabledEnvVar)
	if envTelemetryEnabled != "" {
		envTelemetryEnabled = strings.ToLower(envTelemetryEnabled)

		switch envTelemetryEnabled {
		case "true", "1":
			telemetryEnabled = true
		case "false", "0":
			telemetryEnabled = false
		default:
			return false, fmt.Errorf(
				"invalid value for %s environment variable: '%s', valid values are 'true' or 'false'",
				TelemetryEnabledEnvVar, envTelemetryEnabled,
			)
		}
	}

	return telemetryEnabled, nil
}

// getBindPort returns the TCP port to bind the mcpjungle server to
// precedence: command line flag > environment variable > default
func getBindPort() string {
	port := startServerCmdBindPort
	if port == "" {
		port = os.Getenv(BindPortEnvVar)
	}
	if port == "" {
		port = BindPortDefault
	}
	return port
}

func runStartServer(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load()

	desiredServerMode, err := getDesiredServerMode()
	if err != nil {
		return err
	}

	// Initialize metrics if enabled
	telemetryEnabled, err := isTelemetryEnabled(desiredServerMode)
	if err != nil {
		return err
	}
	otelConfig := &telemetry.Config{
		ServiceName: "mcpjungle",
		Enabled:     telemetryEnabled,
	}
	otelProviders, err := telemetry.Init(cmd.Context(), otelConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize Opentelemetry providers: %v", err)
	}
	defer func() {
		if err := otelProviders.Shutdown(cmd.Context()); err != nil {
			cmd.Printf("Warning: failed to shutdown opentelemetry providers: %v\n", err)
		}
	}()

	// Create MCP metrics from the metrics providers
	// By default, a no-op metrics implementation is used, assuming metrics are disabled.
	// If metrics are enabled, then create the real metrics implementation.
	// This way, we don't have to check if metrics are enabled every time we want to record a metric.
	// Instead, the no-op implementation will simply do nothing.
	// This also avoids nil pointer dereferences in case metrics are not initialized.
	// The rest of the code can simply use the CustomMetrics interface without worrying about whether
	// metrics are enabled or not.
	mcpMetrics := telemetry.NewNoopCustomMetrics()
	if otelProviders.IsEnabled() {
		mcpMetrics, err = telemetry.NewOtelCustomMetrics(otelProviders.Meter)
		if err != nil {
			return fmt.Errorf("failed to create MCP metrics: %v", err)
		}
	}

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

	bindPort := getBindPort()

	// create the MCP proxy server
	mcpProxyServer := server.NewMCPServer(
		"MCPJungle Proxy MCP Server",
		"0.0.1",
		server.WithToolCapabilities(true),
	)

	mcpService, err := mcp.NewMCPService(dbConn, mcpProxyServer, mcpMetrics)
	if err != nil {
		return fmt.Errorf("failed to create MCP service: %v", err)
	}

	mcpClientService := mcpclient.NewMCPClientService(dbConn)

	configService := config.NewServerConfigService(dbConn)
	userService := user.NewUserService(dbConn)

	toolGroupService, err := toolgroup.NewToolGroupService(dbConn, mcpService)
	if err != nil {
		return fmt.Errorf("failed to create Tool Group service: %v", err)
	}

	// create the API server
	opts := &api.ServerOptions{
		Port:             bindPort,
		MCPProxyServer:   mcpProxyServer,
		MCPService:       mcpService,
		MCPClientService: mcpClientService,
		ConfigService:    configService,
		UserService:      userService,
		ToolGroupService: toolGroupService,
		OtelProviders:    otelProviders,
		Metrics:          mcpMetrics,
	}
	s, err := api.NewServer(opts)
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
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
		if desiredServerMode != mode {
			return fmt.Errorf(
				"server is already initialized in %s mode, cannot start in %s mode",
				mode, desiredServerMode,
			)
		}
	} else {
		// If server isn't already initialized and the desired mode is dev, silently initialize the server.
		// Individual (dev mode) users need not worry about server initialization.
		if desiredServerMode == model.ModeDev {
			if err := s.InitDev(); err != nil {
				return fmt.Errorf("failed to initialize server in development mode: %v", err)
			}
		} else {
			// If desired mode is prod, then server initialization is a manual next step to be taken by the user.
			// This is so that they can obtain the admin access token on their client machine.
			cmd.Println(
				"Starting server in Production mode," +
					" don't forget to initialize it by running the `init-server` command",
			)
		}
	}

	// Display startup banner when the server is started
	cmd.Print(asciiArt)
	cmd.Printf("MCPJungle HTTP server listening on :%s\n\n", bindPort)
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to run the server: %v", err)
	}

	return nil
}
