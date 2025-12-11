package cmd

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
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

const (
	PostgresHostEnvVar     = "POSTGRES_HOST"
	PostgresPortEnvVar     = "POSTGRES_PORT"
	PostgresUserEnvVar     = "POSTGRES_USER"
	PostgresPasswordEnvVar = "POSTGRES_PASSWORD"
	PostgresDBEnvVar       = "POSTGRES_DB"
)

const (
	// McpServerInitReqTimeoutSecEnvVar is the environment variable for configuring
	// the MCP server initialization request timeout.
	McpServerInitReqTimeoutSecEnvVar = "MCP_SERVER_INIT_REQ_TIMEOUT_SEC"

	// McpServerInitRequestTimeoutSecondsDefault is the default timeout in seconds for MCP server
	// initialization requests.
	McpServerInitRequestTimeoutSecondsDefault = 10
)

var (
	startServerCmdBindPort          string
	startServerCmdEnterpriseEnabled bool
	startServerCmdProdEnabled       bool
)

var startServerCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the MCPJungle server",
	Long: "Starts the MCPJungle HTTP Registry and the MCP Gateway\n\n" +
		"The server is started in development mode by default, which is ideal for running mcpjungle locally.\n" +
		"Teams & Enterprises should run mcpjungle in enterprise mode.\n\n" +
		"By default, this command creates a SQLite database file in the current directory (if it doesn't already exist).\n" +
		"You can also supply a custom DSN in the DATABASE_URL environment variable.\n" +
		"eg: export DATABASE_URL='postgres://user:password@localhost:5432/mcpjungle'\n" +
		"For Postgres, you can also set individual connection details using the following environment variables:\n" +
		"POSTGRES_HOST, POSTGRES_PORT (default 5432), POSTGRES_USER (default postgres), POSTGRES_PASSWORD, POSTGRES_DB (default postgres)\n",
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
		&startServerCmdEnterpriseEnabled,
		"enterprise",
		false,
		fmt.Sprintf(
			"Run the server in Enterprise mode (ideal for teams and enterprises)."+
				" Alternatively, set the %s environment variable ('%s' | '%s')",
			ServerModeEnvVar, model.ModeDev, model.ModeEnterprise,
		),
	)
	startServerCmd.Flags().BoolVar(
		&startServerCmdProdEnabled,
		"prod",
		false,
		"[DEPRECATED] Alias for --enterprise flag.",
	)

	rootCmd.AddCommand(startServerCmd)
}

// getDesiredServerMode returns the desired server mode for mcpjungle server.
// unless explicitly specified, the desired mode is dev
func getDesiredServerMode(cmd *cobra.Command) (model.ServerMode, error) {
	desiredServerMode := model.ModeDev

	envMode := os.Getenv(ServerModeEnvVar)
	if envMode != "" {
		// the value of the environment variable is allowed to be case-insensitive
		envMode = strings.ToLower(envMode)

		// If user is using the deprecated 'production' mode, replace it with 'enterprise'
		if envMode == string(model.ModeProd) {
			cmd.Printf(
				"Warning: '%s' value is deprecated for env var %s, please use '%s' instead\n\n",
				model.ModeProd, ServerModeEnvVar, model.ModeEnterprise,
			)
			envMode = string(model.ModeEnterprise)
		}

		if envMode != string(model.ModeDev) && envMode != string(model.ModeEnterprise) {
			return "", fmt.Errorf(
				"invalid value for %s environment variable: '%s', valid values are '%s' and '%s'",
				ServerModeEnvVar, envMode, model.ModeDev, model.ModeEnterprise,
			)
		}

		desiredServerMode = model.ServerMode(envMode)
	}

	// If the --enterprise or --prod flag is set, it gets precedence over the environment variable
	if startServerCmdEnterpriseEnabled || startServerCmdProdEnabled {
		desiredServerMode = model.ModeEnterprise
	}
	if startServerCmdProdEnabled {
		cmd.Println("Warning: --prod flag is deprecated, please use --enterprise flag instead")
	}

	return desiredServerMode, nil
}

// isTelemetryEnabled returns true if telemetry should be enabled.
// If an env var is specified, it takes precedence over the defaults.
// Otherwise, by default, telemetry is disabled in dev mode and enabled in enterprise mode.
func isTelemetryEnabled(desiredServerMode model.ServerMode) (bool, error) {
	telemetryEnabled := desiredServerMode == model.ModeEnterprise

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

// getEnvOrFile returns the value of the given environment variable.
// If the environment variable is not set, it checks for a corresponding
// _FILE environment variable and reads the value from the file if it exists.
// If neither is set, it returns an empty string.
// If both are set, the value of the original environment variable takes precedence.
func getEnvOrFile(envVar string) (string, error) {
	val := os.Getenv(envVar)
	if val != "" {
		return val, nil
	}

	fileEnvVar := envVar + "_FILE"
	filePath := os.Getenv(fileEnvVar)
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", fileEnvVar, err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	return "", nil
}

// getPostgresDSN constructs a Postgres DSN from individual Postgres-specific environment variables & files.
// It is used to provide an alternative way to specify Postgres connection details
// in case the user doesn't want to use a full DATABASE_URL.
// If POSTGRES_HOST is not set, this function assumes that Postgres-specific env vars are not being used
// and returns ok=false.
// Other Postgres env vars are optional and have sensible defaults.
func getPostgresDSN() (string, bool, error) {
	host := os.Getenv(PostgresHostEnvVar)
	if host == "" {
		return "", false, nil
	}
	port := os.Getenv(PostgresPortEnvVar)
	if port == "" {
		port = "5432"
	}
	dbName, err := getEnvOrFile(PostgresDBEnvVar)
	if err != nil {
		return "", false, fmt.Errorf("failed to get postgres DB name: %w", err)
	}
	if dbName == "" {
		dbName = "postgres"
	}
	pgUser, err := getEnvOrFile(PostgresUserEnvVar)
	if err != nil {
		return "", false, fmt.Errorf("failed to get postgres user: %w", err)
	}
	if pgUser == "" {
		pgUser = "postgres"
	}
	password, err := getEnvOrFile(PostgresPasswordEnvVar)
	if err != nil {
		return "", false, fmt.Errorf("failed to get postgres password: %w", err)
	}
	// password can be empty, so no default value

	// todo: support sslmode param in the dsn constructed here
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		url.QueryEscape(pgUser),
		url.QueryEscape(password),
		host,
		port,
		url.QueryEscape(dbName),
	)

	return dsn, true, nil
}

// getMcpServerInitReqTimeout returns the timeout (in seconds) for MCP server initialization requests.
// If the corresponding environment variable is not set, it returns the default value.
// If the value is invalid, it returns an error.
func getMcpServerInitReqTimeout() (int, error) {
	timeoutStr := strings.TrimSpace(os.Getenv(McpServerInitReqTimeoutSecEnvVar))
	if timeoutStr == "" {
		return McpServerInitRequestTimeoutSecondsDefault, nil
	}
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil || timeout < 1 {
		return 0, fmt.Errorf(
			"invalid value for %s: '%s', must be a positive integer", McpServerInitReqTimeoutSecEnvVar, timeoutStr,
		)
	}
	return timeout, nil
}

func runStartServer(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load()

	desiredServerMode, err := getDesiredServerMode(cmd)
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

	if dsn == "" {
		// If DATABASE_URL isn't set, try to construct a Postgres DSN if postgres-specific env vars are set.
		pgDSN, ok, err := getPostgresDSN()
		if err != nil {
			return fmt.Errorf("failed to get postgres DSN: %w", err)
		}
		if ok {
			dsn = pgDSN
		}
	}

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

	// create the MCP proxy servers
	mcpProxyServer := server.NewMCPServer(
		"MCPJungle Proxy MCP Server",
		"0.0.1",
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
	)
	sseMcpProxyServer := server.NewMCPServer(
		"MCPJungle Proxy MCP Server for SSE transport",
		"0.0.1",
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
	)

	timeout, err := getMcpServerInitReqTimeout()
	if err != nil {
		return err
	}
	log.Printf("[server] timeout for initialization requests to MCP servers is %d seconds\n", timeout)

	mcpServiceConfig := &mcp.ServiceConfig{
		Db:                      dbConn,
		McpProxyServer:          mcpProxyServer,
		SseMcpProxyServer:       sseMcpProxyServer,
		Metrics:                 mcpMetrics,
		McpServerInitReqTimeout: timeout,
	}
	mcpService, err := mcp.NewMCPService(mcpServiceConfig)
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
		Port:              bindPort,
		MCPProxyServer:    mcpProxyServer,
		SseMcpProxyServer: sseMcpProxyServer,
		MCPService:        mcpService,
		MCPClientService:  mcpClientService,
		ConfigService:     configService,
		UserService:       userService,
		ToolGroupService:  toolGroupService,
		OtelProviders:     otelProviders,
		Metrics:           mcpMetrics,
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
			// If desired mode is enterprise, then server initialization is a manual next step to be taken by the user.
			// This is so that they can obtain the admin access token on their client machine.
			cmd.Println(
				"Starting server in Enterprise mode," +
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
