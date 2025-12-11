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

// ServerMetadata represents the server metadata response
type ServerMetadata struct {
	Version string `json:"version"`
}

// EnableDisableServerResult represents the result of enabling or disabling an MCP server
type EnableDisableServerResult struct {
	// Name is the name of the server that was enabled/disabled
	Name string `json:"name"`
	// ToolsAffected is the number of tools that were enabled/disabled as a result of this operation
	ToolsAffected []string `json:"tools_affected"`
	// PromptsAffected is the number of prompts that were enabled/disabled as a result of this operation
	PromptsAffected []string `json:"prompts_affected"`
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
