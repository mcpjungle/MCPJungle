package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/mcpjungle/mcpjungle/client"
	"github.com/mcpjungle/mcpjungle/internal/configresolver"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

const oauthCallbackPath = "/oauth/callback"

var (
	registerCmdServerName  string
	registerCmdServerURL   string
	registerCmdServerDesc  string
	registerCmdBearerToken string

	registerCmdServerConfigFilePath string
	registerCmdForce                bool
)

var registerMCPServerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register an MCP Server",
	Long: "Register an MCP Server in mcpjungle.\n" +
		"The recommended way is to specify the json configuration file for your mcp server.\n" +
		"Flags are provided for convenience if you want to register a streamable http based server.\n" +
		"But a config file is *required* if you want to register a server using stdio or sse transport.\n" +
		"\nNOTE: A server's name is unique across mcpjungle and must not contain\nany whitespaces, special characters or multiple consecutive underscores '__'.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip flag validation if config file is provided
		if registerCmdServerConfigFilePath != "" {
			return nil
		}
		// Otherwise, validate required flags
		if registerCmdServerName == "" {
			return fmt.Errorf("either supply a configuration file or set the required flag \"name\"")
		}
		if registerCmdServerURL == "" {
			return fmt.Errorf("required flag \"url\" not set")
		}
		return nil
	},
	RunE: runRegisterMCPServer,
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "2",
	},
}

func init() {
	registerMCPServerCmd.Flags().StringVar(
		&registerCmdServerName,
		"name",
		"",
		"MCP server name",
	)
	registerMCPServerCmd.Flags().StringVar(
		&registerCmdServerURL,
		"url",
		"",
		"URL of the streamable http MCP server (eg- http://localhost:8000/mcp)",
	)
	registerMCPServerCmd.Flags().StringVar(
		&registerCmdServerDesc,
		"description",
		"",
		"Server description",
	)
	registerMCPServerCmd.Flags().StringVar(
		&registerCmdBearerToken,
		"bearer-token",
		"",
		"If provided, MCPJungle will use this token to authenticate with the http MCP server for all requests."+
			" This is useful if the MCP server requires static tokens (eg- your API token) for authentication.",
	)
	registerMCPServerCmd.Flags().BoolVar(
		&registerCmdForce,
		"force",
		false,
		"Forcefully register the server even if a server with the same name already exists. This will de-register the existing server, then register the new one.",
	)

	registerMCPServerCmd.Flags().StringVarP(
		&registerCmdServerConfigFilePath,
		"conf",
		"c",
		"",
		"Path to a JSON configuration file for the MCP server.\n"+
			"If provided, the mcp server will be registered using the configuration in the file.\n"+
			"All other flags will be ignored.",
	)

	rootCmd.AddCommand(registerMCPServerCmd)
}

func runRegisterMCPServer(cmd *cobra.Command, args []string) error {
	var input types.RegisterServerInput

	if registerCmdServerConfigFilePath == "" {
		// If no config file is provided, use the flags to create the input for server registration
		input = types.RegisterServerInput{
			Name:        registerCmdServerName,
			Transport:   string(types.TransportStreamableHTTP),
			URL:         registerCmdServerURL,
			Description: registerCmdServerDesc,
			BearerToken: registerCmdBearerToken,
		}
	} else {
		// If a config file is provided, read the configuration from the file
		var err error
		input, err = readMcpServerConfig(registerCmdServerConfigFilePath)
		if err != nil {
			return err
		}
	}

	var callbackSrv *oauthCallbackServer
	result, err := apiClient.RegisterServer(&input, registerCmdForce)
	if err != nil && shouldRetryRegisterWithOAuthCallback(err, &input) {
		// server registration failed because the server requires oauth.
		// Start the local callback server and retry registration with the callback URI included.
		callbackSrv, err = newOAuthCallbackServer()
		if err != nil {
			return fmt.Errorf("failed to start local OAuth callback server: %w", err)
		}
		defer callbackSrv.Close()
		input.OAuthRedirectURI = callbackSrv.RedirectURI()

		result, err = apiClient.RegisterServer(&input, registerCmdForce)
	}
	if err != nil {
		// server registration failed due to an unexpected reason.
		return fmt.Errorf("failed to register server: %w", err)
	}

	if result.AuthorizationRequired != nil {
		if callbackSrv == nil {
			return fmt.Errorf("upstream OAuth authorization required. Open this URL to continue: %s", result.AuthorizationRequired.AuthorizationURL)
		}
		cmd.Printf("OAuth authorization required. Opening browser for upstream server approval.\n")
		openBrowser(result.AuthorizationRequired.AuthorizationURL)

		timeout := time.Until(result.AuthorizationRequired.ExpiresAt)
		if timeout <= 0 {
			timeout = 5 * time.Minute
		}
		params, err := callbackSrv.Wait(timeout)
		if err != nil {
			return fmt.Errorf("failed waiting for OAuth callback: %w", err)
		}

		code := params["code"]
		state := params["state"]
		if code == "" || state == "" {
			return fmt.Errorf("OAuth callback did not include both code and state")
		}

		result, err = apiClient.CompleteUpstreamOAuthSession(
			result.AuthorizationRequired.SessionID,
			&types.CompleteUpstreamOAuthSessionInput{
				Code:  code,
				State: state,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to complete upstream OAuth registration: %w", err)
		}
	}

	if result.Server == nil {
		return fmt.Errorf("server registration completed without a server payload")
	}

	s := result.Server
	fmt.Printf("Server %s registered successfully!\n", s.Name)
	return printRegisteredServerSummary(cmd, s)
}

func readMcpServerConfig(filePath string) (types.RegisterServerInput, error) {
	var input types.RegisterServerInput

	data, err := os.ReadFile(filePath)
	if err != nil {
		return input, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}
	// Parse JSON config
	if err := json.Unmarshal(data, &input); err != nil {
		return input, fmt.Errorf("failed to parse config file: %w", err)
	}
	if err := configresolver.ResolveEnvVars(&input); err != nil {
		return input, fmt.Errorf("failed to resolve config file environment variables: %w", err)
	}

	return input, nil
}

// printRegisteredServerSummary prints the same capability summary for both
// immediate registrations and OAuth-completed registrations.
func printRegisteredServerSummary(cmd *cobra.Command, s *types.McpServer) error {
	if types.McpServerTransport(s.Transport) == types.TransportSSE {
		cmd.Println()
		cmd.Println("This MCP server uses the SSE (Server-sent events) transport.")
		cmd.Println("So its tools will be accessible at the '/sse' endpoint")
		cmd.Println("WARNING: SSE is deprecated, consider migrating this MCP server to streamable http transport.")
	}

	tools, err := apiClient.ListTools(s.Name)
	toolsFetched := err == nil
	if err != nil {
		tools = nil
	}

	prompts, err := apiClient.ListPrompts(s.Name)
	promptsFetched := err == nil
	if err != nil {
		prompts = nil
	}

	resources, err := apiClient.ListResources(s.Name)
	resourcesFetched := err == nil
	if err != nil {
		resources = nil
	}

	printedSection := false
	if len(tools) > 0 {
		cmd.Println()
		cmd.Println("The following tools are now available from this server:")
		for i, tool := range tools {
			cmd.Printf("%d. %s: %s\n\n", i+1, tool.Name, tool.Description)
		}
		printedSection = true
	}

	if len(prompts) > 0 {
		cmd.Println()
		cmd.Println("The following prompts are now available from this server:")
		for i, prompt := range prompts {
			cmd.Printf("%d. %s\n", i+1, prompt.Name)
			if prompt.Description != "" {
				cmd.Printf("   %s\n", prompt.Description)
			}
			cmd.Println()
		}
		printedSection = true
	}

	if len(resources) > 0 {
		cmd.Println()
		cmd.Println("The following resources are now available from this server:")
		for i, resource := range resources {
			cmd.Printf("%d. %s\n", i+1, resource.Name)
			cmd.Printf("   URI: %s\n", resource.URI)
		}
		printedSection = true
	}

	if !printedSection && toolsFetched && promptsFetched && resourcesFetched {
		cmd.Println()
		cmd.Println("This server does not provide any tools, prompts or resources.")
	}

	return nil
}

// shouldRetryRegisterWithOAuthCallback decides whether the CLI should
// start its localhost OAuth callback listener and retry registration after the
// first attempt proved that the upstream server requires OAuth.
func shouldRetryRegisterWithOAuthCallback(err error, input *types.RegisterServerInput) bool {
	if err == nil {
		return false
	}
	var apiErr *client.APIError
	return shouldAutoStartOAuthCallback(input) &&
		errors.As(err, &apiErr) &&
		apiErr.Code == apierrors.CodeUpstreamOAuthRequired
}

// shouldAutoStartOAuthCallback decides whether the CLI should automatically
// provision its localhost callback listener for this registration attempt.
func shouldAutoStartOAuthCallback(input *types.RegisterServerInput) bool {
	if input.OAuthRedirectURI != "" {
		return false
	}
	switch types.McpServerTransport(input.Transport) {
	case types.TransportStreamableHTTP, types.TransportSSE:
		return true
	default:
		return false
	}
}

type oauthCallbackServer struct {
	listener net.Listener
	server   *http.Server
	paramsCh chan map[string]string
}

// newOAuthCallbackServer starts a loopback HTTP server used by the CLI to
// receive OAuth authorization codes from the operator's browser.
func newOAuthCallbackServer() (*oauthCallbackServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	srv := &oauthCallbackServer{
		listener: listener,
		paramsCh: make(chan map[string]string, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(oauthCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		params := map[string]string{}
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}
		select {
		case srv.paramsCh <- params:
		default:
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><h1>Authorization successful</h1><p>You can close this window.</p></body></html>`))
	})

	srv.server = &http.Server{Handler: mux}
	go func() {
		_ = srv.server.Serve(listener)
	}()
	return srv, nil
}

// RedirectURI returns the callback URI that should be supplied during registration.
func (s *oauthCallbackServer) RedirectURI() string {
	return "http://" + s.listener.Addr().String() + oauthCallbackPath
}

// Wait blocks until the callback is received or the timeout elapses.
func (s *oauthCallbackServer) Wait(timeout time.Duration) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case params := <-s.paramsCh:
		return params, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close shuts down the local OAuth callback server.
func (s *oauthCallbackServer) Close() error {
	if s.server == nil {
		return nil
	}
	return s.server.Close()
}

// openBrowser best-effort launches the default browser on the local machine.
func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil {
		fmt.Printf("Open this URL in your browser to continue OAuth authorization:\n%s\n", url)
	}
}
