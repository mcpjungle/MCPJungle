package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// serverToolNameSep is the separator used to combine server name and tool name.
	// This combination produces the canonical name that uniquely identifies a tool across MCPJungle.
	serverToolNameSep = "__"

	// serverPromptNameSep is the separator used to combine server name and prompt name.
	// This combination produces the canonical name that uniquely identifies a prompt across MCPJungle.
	serverPromptNameSep = "__"
)

// Only allow letters, numbers, hyphens, and underscores
var validServerName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateServerName checks if the server name is valid.
// Server name must not contain double underscores `__`.
// Tools in mcpjungle are identified by `<server_name>__<tool_name>` (eg- `github__git_commit`)
// When a tool is invoked, the text before the first __ is treated as the server name.
// eg- In `aws__ec2__create_sg`, `aws` is the MCP server's name and `ec2__create_sg` is the tool.
func validateServerName(name string) error {
	if name == "" {
		return fmt.Errorf("invalid server name: '%s' must not be empty", name)
	}
	if !validServerName.MatchString(name) {
		return fmt.Errorf("invalid server name: '%s' must follow the regular expression %s", name, validServerName)
	}
	if strings.Contains(name, serverToolNameSep) {
		return fmt.Errorf("invalid server name: '%s' must not contain multiple consecutive underscores", name)
	}
	if strings.HasSuffix(name, string(serverToolNameSep[0])) {
		// Don't allow a trailing underscore in server name.
		// This avoids situations like this: `aws_` + `ec2_create_sg` -> `aws___ec2_create_sg`
		//  splitting this would result in: `aws` + `_ec2_create_sg` because we always split on
		//  the first occurrence of `__`
		return fmt.Errorf("invalid server name: '%s' must not end with an underscore", name)
	}
	return nil
}

// mergeServerToolNames combines the server name and tool name into a single tool name unique across the registry.
func mergeServerToolNames(s, t string) string {
	return s + serverToolNameSep + t
}

// splitServerToolName splits the unique tool name into server name and tool name.
func splitServerToolName(name string) (string, string, bool) {
	return strings.Cut(name, serverToolNameSep)
}

// mergeServerPromptNames combines the server name and prompt name into a single prompt name unique across the registry.
func mergeServerPromptNames(s, p string) string {
	return s + serverPromptNameSep + p
}

// splitServerPromptName splits the unique prompt name into server name and prompt name.
func splitServerPromptName(name string) (string, string, bool) {
	return strings.Cut(name, serverPromptNameSep)
}

// isLoopbackURL returns true if rawURL resolves to a loopback address.
// It assumes that rawURL is a valid URL.
func isLoopbackURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false // invalid URL, cannot determine loopback
	}
	host := u.Hostname()

	if host == "" {
		return false // no host, not a loopback
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}

	return false
}

// convertToolModelToMcpObject converts a tool model from the database to a mcp.Tool object.
// The returned tool always has a valid InputSchema (defaults to {"type":"object"} if missing).
func convertToolModelToMcpObject(t *model.Tool) (mcp.Tool, error) {
	inputSchema := json.RawMessage(t.InputSchema)
	if len(inputSchema) == 0 {
		inputSchema = json.RawMessage(`{"type":"object"}`)
	} else if !json.Valid(inputSchema) {
		return mcp.Tool{}, fmt.Errorf("invalid InputSchema for tool %s: not valid JSON", t.Name)
	}

	mcpTool := mcp.Tool{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: inputSchema,
	}

	// Restore annotations if present
	if len(t.Annotations) > 0 {
		var annotations mcp.ToolAnnotations
		if err := json.Unmarshal(t.Annotations, &annotations); err != nil {
			// Log the error but don't fail - annotations are optional
			log.Printf("[WARN] failed to unmarshal annotations for tool %s: %v", t.Name, err)
		} else {
			mcpTool.Annotations = &annotations
		}
	}

	// NOTE: if more fields are added to the tool in DB, they should be set here as well

	return mcpTool, nil
}

// convertPromptModelToMcpObject converts a prompt model from the database to a mcp.Prompt object.
func convertPromptModelToMcpObject(p *model.Prompt) (mcp.Prompt, error) {
	mcpPrompt := mcp.Prompt{
		Name:        p.Name,
		Description: p.Description,
	}

	if len(p.Arguments) > 0 {
		var arguments []*mcp.PromptArgument
		if err := json.Unmarshal(p.Arguments, &arguments); err != nil {
			return mcp.Prompt{}, fmt.Errorf(
				"failed to unmarshal arguments %s for prompt %s: %w", p.Arguments, p.Name, err,
			)
		}
		mcpPrompt.Arguments = arguments
	}

	return mcpPrompt, nil
}

// headerRoundTripper is a custom http.RoundTripper that injects HTTP headers
// into every request. It wraps an existing RoundTripper (or http.DefaultTransport).
type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	base := h.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// prepareSHTTPClientHeaders prepares the HTTP headers for a streamable HTTP
// MCP server connection based on the server's configuration.
// If a bearer token is provided and a custom Authorization header is set, the custom header
// takes precedence and the bearer token is ignored.
func prepareSHTTPClientHeaders(serverName string, conf *model.StreamableHTTPConfig) map[string]string {
	headers := map[string]string{}
	for key, value := range conf.Headers {
		headers[key] = value
	}

	if conf.BearerToken != "" {
		if _, hasAuthorizationHeader := headers["Authorization"]; hasAuthorizationHeader {
			log.Printf("[INFO] custom Authorization header will be used for MCP server %s; bearer_token ignored", serverName)
		} else {
			headers["Authorization"] = "Bearer " + conf.BearerToken
		}
	}

	if len(headers) == 0 {
		return nil
	}

	return headers
}

// createHTTPMcpServerConn creates a new connection with a streamable http MCP server and returns the session.
func createHTTPMcpServerConn(ctx context.Context, s *model.McpServer, initReqTimeoutSec int) (*mcp.ClientSession, error) {
	conf, err := s.GetStreamableHTTPConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get streamable HTTP config for MCP server %s: %w", s.Name, err)
	}

	headers := prepareSHTTPClientHeaders(s.Name, conf)

	var httpClient *http.Client
	if len(headers) > 0 {
		httpClient = &http.Client{Transport: &headerRoundTripper{headers: headers}}
	}

	transport := &mcp.StreamableClientTransport{
		Endpoint:   conf.URL,
		HTTPClient: httpClient,
	}

	c := mcp.NewClient(&mcp.Implementation{
		Name:    "mcpjungle mcp client for " + conf.URL,
		Version: "0.1",
	}, nil)

	initCtx, cancel := context.WithTimeout(ctx, time.Duration(initReqTimeoutSec)*time.Second)
	defer cancel()

	session, err := c.Connect(initCtx, transport, nil)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("initialization request to MCP server timed out after %d seconds", initReqTimeoutSec)
		}
		if errors.Is(err, syscall.ECONNREFUSED) && isLoopbackURL(conf.URL) {
			return nil, fmt.Errorf(
				"connection to the MCP server %s was refused. "+
					"If mcpjungle is running inside Docker, use 'host.docker.internal' as your MCP server's hostname",
				conf.URL,
			)
		}
		return nil, fmt.Errorf("failed to initialize connection with MCP server: %w", err)
	}

	return session, nil
}

// captureStdioServerStderr captures the stderr output of a stdio MCP server in the background
// and writes it to mcpjungle server logs.
// This is useful for troubleshooting and visibility into the stdio server's behaviour.
func captureStdioServerStderr(name string, stderr io.Reader) {
	go func() {
		buf := make([]byte, 4096) // 4KB buffer for reading stderr
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				if err == io.EOF || errors.Is(err, os.ErrClosed) {
					log.Printf("['%s' MCP Server] [DEBUG] server process has exited gracefully", name)
				} else {
					log.Printf("['%s' MCP STDERR] Error reading stderr: %v", name, err)
				}
				log.Printf("['%s' MCP server] [DEBUG] exiting goroutine", name)
				break
			}
			if n > 0 {
				log.Printf("['%s' MCP STDERR] %s", name, string(buf[:n]))
			}
		}
	}()
}

// runStdioServer runs a stdio MCP server and returns the session.
func runStdioServer(ctx context.Context, s *model.McpServer, initReqTimeoutSec int) (*mcp.ClientSession, error) {
	conf, err := s.GetStdioConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdio config for MCP server %s: %w", s.Name, err)
	}

	// Build environment: inherit current process env plus config-specific vars
	envVars := os.Environ()
	for k, v := range conf.Env {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Command(conf.Command, conf.Args...)
	cmd.Env = envVars

	// Set up stderr pipe BEFORE Connect() calls cmd.Start() internally
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe for MCP server: %w", err)
	}

	transport := &mcp.CommandTransport{Command: cmd}

	c := mcp.NewClient(&mcp.Implementation{
		Name:    "mcpjungle mcp client for stdio",
		Version: "0.1",
	}, nil)

	// currently, we only capture the stderr output in the mcpjungle server logs.
	// TODO: Propagate the stderr output to the client as well to provide them quicker feedback on errors.
	captureStdioServerStderr(s.Name, stderr)

	initCtx, cancel := context.WithTimeout(ctx, time.Duration(initReqTimeoutSec)*time.Second)
	defer cancel()

	session, err := c.Connect(initCtx, transport, nil)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf(
				"initialization request to MCP server timed out after %d seconds,"+
					" check mcpjungle server logs for any errors from this MCP server",
				initReqTimeoutSec,
			)
		}
		return nil, fmt.Errorf("failed to initialize connection with MCP server: %w", err)
	}

	return session, nil
}

// createSSEMcpServerConn creates a new connection with an SSE transport-based MCP server and returns the session.
func createSSEMcpServerConn(ctx context.Context, s *model.McpServer) (*mcp.ClientSession, error) {
	conf, err := s.GetSSEConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get SSE transport config for MCP server %s: %w", s.Name, err)
	}

	var httpClient *http.Client
	if conf.BearerToken != "" {
		httpClient = &http.Client{
			Transport: &headerRoundTripper{
				headers: map[string]string{"Authorization": "Bearer " + conf.BearerToken},
			},
		}
	}

	transport := &mcp.SSEClientTransport{
		Endpoint:   conf.URL,
		HTTPClient: httpClient,
	}

	c := mcp.NewClient(&mcp.Implementation{
		Name:    "mcpjungle-sse-proxy-client",
		Version: "0.1.0",
	}, nil)

	session, err := c.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("client failed to initialize connection with SSE MCP server: %w", err)
	}

	return session, nil
}

func newMcpServerSession(ctx context.Context, s *model.McpServer, initReqTimeoutSec int) (*mcp.ClientSession, error) {
	if s.Transport == types.TransportStreamableHTTP {
		mcpClient, err := createHTTPMcpServerConn(ctx, s, initReqTimeoutSec)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create connection to streamable http MCP server %s: %w", s.Name, err,
			)
		}
		return mcpClient, nil
	}

	if s.Transport == types.TransportSSE {
		mcpClient, err := createSSEMcpServerConn(ctx, s)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create connection to SSE MCP server %s: %w", s.Name, err,
			)
		}
		return mcpClient, nil
	}

	// A new sub-process is spun up for each call to a STDIO mcp server.
	// This is especially a problem for the MCP proxy server, which is expected to call tools frequently.
	// This causes a serious performance hit, but is easy to implement so it is used for now.
	// For stateful sessions, use the SessionManager to keep the process running.
	mcpClient, err := runStdioServer(ctx, s, initReqTimeoutSec)
	if err != nil {
		return nil, fmt.Errorf("failed to run stdio MCP server %s: %w", s.Name, err)
	}
	return mcpClient, nil
}
