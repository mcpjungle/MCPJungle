package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type StreamableHTTPConfig struct {
	// URL must be a valid http/https URL.
	URL string `json:"url"`

	// TODO: Store the bearer token in a more secure way, e.g., encrypted instead of plaintext.
	// BearerToken is an optional token used for authenticating requests to the MCP server.
	// If present, it will be used to set the Authorization header in all requests to this MCP server.
	BearerToken string `json:"bearer_token,omitempty"`

	// Headers are optional custom HTTP headers forwarded to the MCP server.
	Headers map[string]string `json:"headers,omitempty"`
}

type StdioConfig struct {
	// Command is the shell command to run the stdio mcp server.
	Command string `json:"command"`

	// Args contains a list of strings that are passed as arguments to the command
	Args []string `json:"args,omitempty"`

	// Env describes the environment variables to pass to the MCP server
	Env map[string]string `json:"env,omitempty"`
}

type SSEConfig struct {
	// URL must be a valid http/https URL.
	URL string `json:"url"`

	BearerToken string `json:"bearer_token,omitempty"`
}

// McpServer represents a MCP server registered in mcpjungle
type McpServer struct {
	gorm.Model

	Name      string                   `json:"name" gorm:"uniqueIndex;not null"`
	Transport types.McpServerTransport `json:"transport" gorm:"type:varchar(30);not null"`

	Description string `json:"description"`

	// Config describes the transport-specific configuration for the MCP server.
	// It contains the JSON representation of either StreamableHTTPConfig or StdioConfig.
	Config datatypes.JSON `json:"config" gorm:"type:jsonb;not null"`

	// SessionMode controls how mcpjungle manages connections to this MCP server.
	// "stateless" (default): Creates a new connection for each tool call.
	// "stateful": Maintains a persistent connection across tool calls.
	SessionMode types.SessionMode `json:"session_mode" gorm:"type:varchar(20);default:'stateless'"`
}

// validateURL checks that rawURL is a well-formed http or https URL.
func validateURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url %q: must be a valid http or https URL", rawURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid url scheme %q: must be http or https", u.Scheme)
	}
	return nil
}

// NewStreamableHTTPServer creates a new MCP server with streamable HTTP transport configuration.
func NewStreamableHTTPServer(name, description, url, bearerToken string, headers map[string]string, sessionMode types.SessionMode) (*McpServer, error) {
	if url == "" {
		return nil, errors.New("url is required for streamable HTTP transport")
	}
	if err := validateURL(url); err != nil {
		return nil, err
	}
	config := StreamableHTTPConfig{
		URL:         url,
		BearerToken: bearerToken,
		Headers:     headers,
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	if sessionMode == "" {
		sessionMode = types.SessionModeStateless
	}
	return &McpServer{
		Name:        name,
		Description: description,
		Transport:   types.TransportStreamableHTTP,
		Config:      configJSON,
		SessionMode: sessionMode,
	}, nil
}

// NewStdioServer creates a new MCP server with stdio transport configuration.
func NewStdioServer(name, description, command string, args []string, env map[string]string, sessionMode types.SessionMode) (*McpServer, error) {
	if command == "" {
		return nil, errors.New("command is required for stdio transport")
	}
	config := StdioConfig{
		Command: command,
		Args:    args,
		Env:     env,
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	if sessionMode == "" {
		sessionMode = types.SessionModeStateless
	}
	return &McpServer{
		Name:        name,
		Description: description,
		Transport:   types.TransportStdio,
		Config:      datatypes.JSON(configJSON),
		SessionMode: sessionMode,
	}, nil
}

// NewSSEServer creates a new MCP server with SSE transport configuration.
func NewSSEServer(name, description, url, bearerToken string, sessionMode types.SessionMode) (*McpServer, error) {
	if url == "" {
		return nil, errors.New("url is required for SSE transport")
	}
	if err := validateURL(url); err != nil {
		return nil, err
	}
	config := SSEConfig{
		URL:         url,
		BearerToken: bearerToken,
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	if sessionMode == "" {
		sessionMode = types.SessionModeStateless
	}
	return &McpServer{
		Name:        name,
		Description: description,
		Transport:   types.TransportSSE,
		Config:      configJSON,
		SessionMode: sessionMode,
	}, nil
}

// GetStreamableHTTPConfig returns the configuration if this is a streamable HTTP server
func (s *McpServer) GetStreamableHTTPConfig() (*StreamableHTTPConfig, error) {
	if s.Transport != types.TransportStreamableHTTP {
		return nil, errors.New("server is not a streamable HTTP transport type")
	}
	var config StreamableHTTPConfig
	if err := json.Unmarshal(s.Config, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetStdioConfig returns the configuration if this is a stdio server
func (s *McpServer) GetStdioConfig() (*StdioConfig, error) {
	if s.Transport != types.TransportStdio {
		return nil, errors.New("server is not a stdio transport type")
	}
	var config StdioConfig
	if err := json.Unmarshal(s.Config, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetSSEConfig returns the configuration if this is an SSE server
func (s *McpServer) GetSSEConfig() (*SSEConfig, error) {
	if s.Transport != types.TransportSSE {
		return nil, errors.New("server is not a SSE transport type")
	}
	var config SSEConfig
	if err := json.Unmarshal(s.Config, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
