package types

import (
	"encoding/json"
	"testing"
)

func TestMcpServerTransport(t *testing.T) {
	t.Run("McpServerTransport constants", func(t *testing.T) {
		if TransportStdio != "stdio" {
			t.Errorf("Expected TransportStdio to be 'stdio', got %s", TransportStdio)
		}
		if TransportStreamableHTTP != "streamable_http" {
			t.Errorf("Expected TransportStreamableHTTP to be 'streamable_http', got %s", TransportStreamableHTTP)
		}
	})

	t.Run("McpServerTransport string conversion", func(t *testing.T) {
		stdioTransport := string(TransportStdio)
		httpTransport := string(TransportStreamableHTTP)

		if stdioTransport != "stdio" {
			t.Errorf("Expected stdioTransport string to be 'stdio', got %s", stdioTransport)
		}
		if httpTransport != "streamable_http" {
			t.Errorf("Expected httpTransport string to be 'streamable_http', got %s", httpTransport)
		}
	})
}

func TestMcpServer(t *testing.T) {
	t.Run("McpServer struct creation", func(t *testing.T) {
		server := McpServer{
			Name:      "test-server",
			Transport: "stdio",
			Command:   "/usr/bin/test-server",
		}

		if server.Name != "test-server" {
			t.Errorf("Expected Name to be 'test-server', got %s", server.Name)
		}
		if server.Transport != "stdio" {
			t.Errorf("Expected Transport to be 'stdio', got %s", server.Transport)
		}
		if server.Command != "/usr/bin/test-server" {
			t.Errorf("Expected Command to be '/usr/bin/test-server', got %s", server.Command)
		}
	})

	t.Run("McpServer JSON marshaling", func(t *testing.T) {
		server := McpServer{
			Name:        "json-server",
			Transport:   "stdio",
			Description: "Server for JSON testing",
			Command:     "/usr/bin/json-server",
			Args:        []string{"--verbose"},
			Env:         map[string]string{"ENV": "test"},
		}

		data, err := json.Marshal(server)
		if err != nil {
			t.Fatalf("Failed to marshal McpServer: %v", err)
		}

		expected := `{"name":"json-server","transport":"stdio","description":"Server for JSON testing","url":"","command":"/usr/bin/json-server","args":["--verbose"],"env":{"ENV":"test"}}`
		if string(data) != expected {
			t.Errorf("Expected JSON %s, got %s", expected, string(data))
		}
	})
}

func TestValidateTransport(t *testing.T) {
	t.Run("ValidateTransport with valid stdio", func(t *testing.T) {
		transport, err := ValidateTransport("stdio")
		if err != nil {
			t.Errorf("Expected no error for 'stdio', got %v", err)
		}
		if transport != TransportStdio {
			t.Errorf("Expected transport to be TransportStdio, got %s", transport)
		}
	})

	t.Run("ValidateTransport with valid streamable_http", func(t *testing.T) {
		transport, err := ValidateTransport("streamable_http")
		if err != nil {
			t.Errorf("Expected no error for 'streamable_http', got %v", err)
		}
		if transport != TransportStreamableHTTP {
			t.Errorf("Expected transport to be TransportStreamableHTTP, got %s", transport)
		}
	})

	t.Run("ValidateTransport with empty string", func(t *testing.T) {
		transport, err := ValidateTransport("")
		if err == nil {
			t.Error("Expected error for empty string, got nil")
		}
		if transport != "" {
			t.Errorf("Expected empty transport for invalid input, got %s", transport)
		}
	})

	t.Run("ValidateTransport with invalid transport", func(t *testing.T) {
		transport, err := ValidateTransport("invalid_transport")
		if err == nil {
			t.Error("Expected error for invalid transport, got nil")
		}
		if transport != "" {
			t.Errorf("Expected empty transport for invalid input, got %s", transport)
		}
	})
}
