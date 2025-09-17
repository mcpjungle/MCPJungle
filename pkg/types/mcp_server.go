package types

import "fmt"

// McpServerTransport represents the transport protocol used by an MCP server.
// All transport types supported by mcpjungle are defined in this file with this type.
type McpServerTransport string

const (
	TransportStdio          McpServerTransport = "stdio"
	TransportStreamableHTTP McpServerTransport = "streamable_http"
	TransportSSE            McpServerTransport = "sse"
)

// McpServer represents an MCP server registered in the MCPJungle registry.
type McpServer struct {
	Name        string `json:"name"`
	Transport   string `json:"transport"`
	Description string `json:"description"`

	URL string `json:"url"`

	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// RegisterServerInput is the input structure for registering a new MCP server with mcpjungle.
// It is also the basis for the JSON configuration file used to register a new MCP server.
type RegisterServerInput struct {
	// Name is the unique name of an MCP server registered in mcpjungle
	Name string `json:"name"`

	// Transport is the transport protocol used by the MCP server.
	// valid values are "stdio", "streamable_http", and "sse".
	Transport string `json:"transport"`

	Description string `json:"description"`

	// URL is the URL of the remote mcp server
	// It is mandatory when transport is streamable_http and must be a valid
	//  http/https URL (e.g., https://example.com/mcp).
	URL string `json:"url"`

	// BearerToken is an optional token used for authenticating requests to the remote MCP server.
	// It is useful when the upstream MCP server requires static tokens (e.g., API tokens) for authentication.
	// If the transport is "stdio", this field is ignored.
	BearerToken string `json:"bearer_token"`

	// Command is the command to run the mcp server.
	// It is mandatory when the transport is "stdio".
	Command string `json:"command"`

	// Args is the list of arguments to pass to the command when the transport is "stdio".
	Args []string `json:"args"`

	// Env is the set of environment variables to pass to the mcp server when the transport is "stdio".
	// Both the key and value must be of type string.
	Env map[string]string `json:"env"`
}

// InstallOptions represents the options for installing an MCP server from the registry.
type InstallOptions struct {
	// ServerName is the name of the server to install from the registry
	ServerName string `json:"server_name"`

	// Args are additional arguments to pass to the MCP server
	Args []string `json:"args,omitempty"`

	// Env are environment variables to set for the MCP server
	Env map[string]string `json:"env,omitempty"`

	// Version is the specific version of the MCP server to install
	Version string `json:"version,omitempty"`
}

// ServerRegistry represents a server entry in the official MCP servers registry.
type ServerRegistry struct {
	// Name is the display name of the server
	Name string `json:"name"`

	// Package is the package name (e.g., "@modelcontextprotocol/server-time")
	Package string `json:"package"`

	// Transport is the transport protocol used by the server
	Transport string `json:"transport"`

	// Command is the command to run the server (e.g., "npx", "uvx")
	Command string `json:"command"`

	// Args are the default arguments for the server
	Args []string `json:"args"`

	// Description is a brief description of what the server does
	Description string `json:"description"`

	// Category is the category the server belongs to (e.g., "utility", "filesystem")
	Category string `json:"category"`

	// PackageManager is the package manager used (e.g., "npm", "pip")
	PackageManager string `json:"package_manager"`

	// RequiredEnvVars lists environment variables that must be set
	RequiredEnvVars []string `json:"required_env_vars,omitempty"`

	// OptionalEnvVars lists optional environment variables with defaults
	OptionalEnvVars map[string]string `json:"optional_env_vars,omitempty"`

	// URL is the URL of the remote MCP server (for HTTP/SSE transports)
	URL string `json:"url,omitempty"`

	// BearerToken is an optional token for authenticating with remote servers
	BearerToken string `json:"bearer_token,omitempty"`
}

// ValidateTransport validates the input string and returns the corresponding model.McpServerTransport.
// It returns an error if the input is invalid or empty.
func ValidateTransport(input string) (McpServerTransport, error) {
	errMsgExt := fmt.Sprintf(
		"(acceptable values: '%s', '%s', '%s')", TransportStreamableHTTP, TransportStdio, TransportSSE,
	)

	switch input {
	case string(TransportStreamableHTTP):
		return TransportStreamableHTTP, nil
	case string(TransportStdio):
		return TransportStdio, nil
	case string(TransportSSE):
		return TransportSSE, nil
	case "":
		return "", fmt.Errorf("transport is required %s", errMsgExt)
	default:
		return "", fmt.Errorf("unsupported transport type: %s %s", input, errMsgExt)
	}
}
