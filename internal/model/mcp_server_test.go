package model

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestNewStreamableHTTPServer(t *testing.T) {
	tests := []struct {
		name        string
		serverName  string
		description string
		url         string
		bearerToken string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid server with bearer token",
			serverName:  "test-server",
			description: "Test MCP server",
			url:         "https://example.com",
			bearerToken: "secret-token",
			wantErr:     false,
		},
		{
			name:        "valid server without bearer token",
			serverName:  "test-server-2",
			description: "Another test server",
			url:         "http://localhost:8080",
			bearerToken: "",
			wantErr:     false,
		},
		{
			name:        "empty url",
			serverName:  "invalid-server",
			description: "Should fail",
			url:         "",
			bearerToken: "token",
			wantErr:     true,
			errMsg:      "url is required for streamable HTTP transport",
		},
		{
			// empty name is tolerated because these methods only validate the transport-specific fields.
			// name validation's responsibility lies with the main mcp registration logic.
			name:        "empty name is allowed",
			serverName:  "",
			description: "Empty name",
			url:         "https://example.com",
			bearerToken: "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewStreamableHTTPServer(tt.serverName, tt.description, tt.url, tt.bearerToken, "")

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if err != nil && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if server == nil {
				t.Errorf("expected server, got nil")
				return
			}

			if server.Name != tt.serverName {
				t.Errorf("expected name %q, got %q", tt.serverName, server.Name)
			}

			if server.Description != tt.description {
				t.Errorf("expected description %q, got %q", tt.description, server.Description)
			}

			if server.Transport != types.TransportStreamableHTTP {
				t.Errorf("expected transport %q, got %q", types.TransportStreamableHTTP, server.Transport)
			}

			config, err := server.GetStreamableHTTPConfig()
			if err != nil {
				t.Errorf("failed to get config: %v", err)
			}

			if config.URL != tt.url {
				t.Errorf("expected URL %q, got %q", tt.url, config.URL)
			}

			if config.BearerToken != tt.bearerToken {
				t.Errorf("expected bearer token %q, got %q", tt.bearerToken, config.BearerToken)
			}
		})
	}
}

func TestNewStdioServer(t *testing.T) {
	tests := []struct {
		name        string
		serverName  string
		description string
		command     string
		args        []string
		env         map[string]string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid server with args and env",
			serverName:  "stdio-server",
			description: "Test stdio server",
			command:     "/usr/bin/python3",
			args:        []string{"script.py", "--debug"},
			env:         map[string]string{"PYTHONPATH": "/app"},
			wantErr:     false,
		},
		{
			name:        "valid server without args and env",
			serverName:  "simple-server",
			description: "Simple stdio server",
			command:     "node",
			args:        nil,
			env:         nil,
			wantErr:     false,
		},
		{
			name:        "valid server with only args",
			serverName:  "args-server",
			description: "Server with args",
			command:     "bash",
			args:        []string{"-c", "echo hello"},
			env:         nil,
			wantErr:     false,
		},
		{
			name:        "empty command",
			serverName:  "invalid-server",
			description: "Should fail",
			command:     "",
			args:        []string{"arg"},
			env:         nil,
			wantErr:     true,
			errMsg:      "command is required for stdio transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewStdioServer(tt.serverName, tt.description, tt.command, tt.args, tt.env, "")

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if err != nil && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if server == nil {
				t.Errorf("expected server, got nil")
				return
			}

			if server.Name != tt.serverName {
				t.Errorf("expected name %q, got %q", tt.serverName, server.Name)
			}

			if server.Description != tt.description {
				t.Errorf("expected description %q, got %q", tt.description, server.Description)
			}

			if server.Transport != types.TransportStdio {
				t.Errorf("expected transport %q, got %q", types.TransportStdio, server.Transport)
			}

			config, err := server.GetStdioConfig()
			if err != nil {
				t.Errorf("failed to get config: %v", err)
			}

			if config.Command != tt.command {
				t.Errorf("expected command %q, got %q", tt.command, config.Command)
			}

			if len(config.Args) != len(tt.args) {
				t.Errorf("expected %d args, got %d", len(tt.args), len(config.Args))
			} else {
				for i, arg := range tt.args {
					if config.Args[i] != arg {
						t.Errorf("expected arg[%d] %q, got %q", i, arg, config.Args[i])
					}
				}
			}

			if len(config.Env) != len(tt.env) {
				t.Errorf("expected %d env vars, got %d", len(tt.env), len(config.Env))
			} else {
				for key, val := range tt.env {
					if config.Env[key] != val {
						t.Errorf("expected env[%q]=%q, got %q", key, val, config.Env[key])
					}
				}
			}
		})
	}
}

func TestNewSSEServer(t *testing.T) {
	tests := []struct {
		name        string
		serverName  string
		description string
		url         string
		bearerToken string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid server with bearer token",
			serverName:  "sse-server",
			description: "Test SSE server",
			url:         "https://example.com/events",
			bearerToken: "secret-token",
			wantErr:     false,
		},
		{
			name:        "valid server without bearer token",
			serverName:  "sse-server-2",
			description: "Another SSE server",
			url:         "http://localhost:3000/stream",
			bearerToken: "",
			wantErr:     false,
		},
		{
			name:        "empty url",
			serverName:  "invalid-server",
			description: "Should fail",
			url:         "",
			bearerToken: "token",
			wantErr:     true,
			errMsg:      "url is required for SSE transport",
		},
		{
			name:        "empty name is allowed",
			serverName:  "",
			description: "Empty name",
			url:         "https://example.com/events",
			bearerToken: "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewSSEServer(tt.serverName, tt.description, tt.url, tt.bearerToken, "")

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if err != nil && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if server == nil {
				t.Errorf("expected server, got nil")
				return
			}

			if server.Name != tt.serverName {
				t.Errorf("expected name %q, got %q", tt.serverName, server.Name)
			}

			if server.Description != tt.description {
				t.Errorf("expected description %q, got %q", tt.description, server.Description)
			}

			if server.Transport != types.TransportSSE {
				t.Errorf("expected transport %q, got %q", types.TransportSSE, server.Transport)
			}

			config, err := server.GetSSEConfig()
			if err != nil {
				t.Errorf("failed to get config: %v", err)
			}

			if config.URL != tt.url {
				t.Errorf("expected URL %q, got %q", tt.url, config.URL)
			}

			if config.BearerToken != tt.bearerToken {
				t.Errorf("expected bearer token %q, got %q", tt.bearerToken, config.BearerToken)
			}
		})
	}
}

func TestNewStdioServerWithSessionMode(t *testing.T) {
	tests := []struct {
		name         string
		sessionMode  types.SessionMode
		expectedMode types.SessionMode
	}{
		{
			name:         "stateful session mode",
			sessionMode:  types.SessionModeStateful,
			expectedMode: types.SessionModeStateful,
		},
		{
			name:         "stateless session mode",
			sessionMode:  types.SessionModeStateless,
			expectedMode: types.SessionModeStateless,
		},
		{
			name:         "empty session mode defaults to stateless",
			sessionMode:  "",
			expectedMode: types.SessionModeStateless,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewStdioServer(
				"test-server",
				"Test description",
				"echo",
				[]string{"hello"},
				nil,
				tt.sessionMode,
			)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if server.SessionMode != tt.expectedMode {
				t.Errorf("expected session mode %q, got %q", tt.expectedMode, server.SessionMode)
			}
		})
	}
}

func TestNewStreamableHTTPServerWithSessionMode(t *testing.T) {
	tests := []struct {
		name         string
		sessionMode  types.SessionMode
		expectedMode types.SessionMode
	}{
		{
			name:         "stateful session mode",
			sessionMode:  types.SessionModeStateful,
			expectedMode: types.SessionModeStateful,
		},
		{
			name:         "stateless session mode",
			sessionMode:  types.SessionModeStateless,
			expectedMode: types.SessionModeStateless,
		},
		{
			name:         "empty session mode defaults to stateless",
			sessionMode:  "",
			expectedMode: types.SessionModeStateless,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewStreamableHTTPServer(
				"test-server",
				"Test description",
				"https://example.com",
				"",
				tt.sessionMode,
			)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if server.SessionMode != tt.expectedMode {
				t.Errorf("expected session mode %q, got %q", tt.expectedMode, server.SessionMode)
			}
		})
	}
}

func TestNewSSEServerWithSessionMode(t *testing.T) {
	tests := []struct {
		name         string
		sessionMode  types.SessionMode
		expectedMode types.SessionMode
	}{
		{
			name:         "stateful session mode",
			sessionMode:  types.SessionModeStateful,
			expectedMode: types.SessionModeStateful,
		},
		{
			name:         "stateless session mode",
			sessionMode:  types.SessionModeStateless,
			expectedMode: types.SessionModeStateless,
		},
		{
			name:         "empty session mode defaults to stateless",
			sessionMode:  "",
			expectedMode: types.SessionModeStateless,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewSSEServer(
				"test-server",
				"Test description",
				"https://example.com/sse",
				"",
				tt.sessionMode,
			)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if server.SessionMode != tt.expectedMode {
				t.Errorf("expected session mode %q, got %q", tt.expectedMode, server.SessionMode)
			}
		})
	}
}
