package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"net"
	"net/url"
	"regexp"
	"strings"
	"syscall"
)

// serverToolNameSep is the separator used to combine server name and tool name.
// This combination produces the canonical name that uniquely identifies a tool across MCPJungle.
const serverToolNameSep = "__"

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

// convertToolModelToMcpObject converts a tool model from the database to a mcp.Tool object
func convertToolModelToMcpObject(t *model.Tool) (mcp.Tool, error) {
	mcpTool := mcp.Tool{
		Name:        t.Name,
		Description: t.Description,
	}

	var inputSchema mcp.ToolInputSchema
	if err := json.Unmarshal(t.InputSchema, &inputSchema); err != nil {
		return mcp.Tool{}, fmt.Errorf(
			"failed to unmarshal input schema %s for tool %s: %w", t.InputSchema, t.Name, err,
		)
	}
	mcpTool.InputSchema = inputSchema

	// TODO: Add other attributes to the tool, such as annotations
	// NOTE: if more fields are added to the tool in DB, they should be set here as well

	return mcpTool, nil
}

// createHTTPMcpServerConn creates a new connection with a streamable http MCP server and returns the client.
func createHTTPMcpServerConn(ctx context.Context, s *model.McpServer) (*client.Client, error) {
	conf, err := s.GetStreamableHTTPConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get streamable HTTP config for MCP server %s: %w", s.Name, err)
	}

	var opts []transport.StreamableHTTPCOption
	if conf.BearerToken != "" {
		// If bearer token is provided, set the Authorization header
		o := transport.WithHTTPHeaders(map[string]string{
			"Authorization": "Bearer " + conf.BearerToken,
		})
		opts = append(opts, o)
	}

	c, err := client.NewStreamableHttpClient(conf.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create streamable HTTP client for MCP server: %w", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "mcpjungle mcp client for " + conf.URL,
		Version: "0.1",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) && isLoopbackURL(conf.URL) {
			return nil, fmt.Errorf(
				"connection to the MCP server %s was refused. "+
					"If mcpjungle is running inside Docker, use 'host.docker.internal' as your MCP server's hostname",
				conf.URL,
			)
		}
		return nil, fmt.Errorf("failed to initialize connection with MCP server: %w", err)
	}

	return c, nil
}

// runStdioServer runs a stdio MCP server and returns the client.
func runStdioServer(ctx context.Context, s *model.McpServer) (*client.Client, error) {
	conf, err := s.GetStdioConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdio config for MCP server %s: %w", s.Name, err)
	}

	// Convert the environment map to a slice of strings in the format "KEY=VALUE"
	envVars := make([]string, 0)
	if conf.Env != nil {
		for k, v := range conf.Env {
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}
	}

	c, err := client.NewStdioMCPClient(conf.Command, envVars, conf.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create stdio client for MCP server: %w", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "mcpjungle mcp client for stdio",
		Version: "0.1",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize connection with MCP server: %w", err)
	}

	return c, nil
}

func newMcpServerSession(ctx context.Context, s *model.McpServer) (*client.Client, error) {
	if s.Transport == types.TransportStreamableHTTP {
		mcpClient, err := createHTTPMcpServerConn(ctx, s)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create connection to streamable http MCP server %s: %w", s.Name, err,
			)
		}
		return mcpClient, nil
	}

	// A new sub-process is spun up for each call to a STDIO mcp server.
	// This is especially a problem for the MCP proxy server, which is expected to call tools frequently.
	// This causes a serious performance hit, but is easy to implement so it is used for now.
	// TODO: Think of a better solution, ie, re-use connections to stdio MCP servers.
	mcpClient, err := runStdioServer(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("failed to run stdio MCP server %s: %w", s.Name, err)
	}
	return mcpClient, nil
}
