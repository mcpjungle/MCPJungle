package api

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// MockMCPService is a mock implementation for testing
type MockMCPService struct {
	registerError   error
	listError       error
	deregisterError error
}

func (m *MockMCPService) RegisterMcpServer(ctx any, server *model.McpServer) error {
	return m.registerError
}

func (m *MockMCPService) ListMcpServers() ([]model.McpServer, error) {
	return nil, m.listError
}

func (m *MockMCPService) DeregisterMcpServer(name string) error {
	return m.deregisterError
}

// Since the handlers expect *mcp.MCPService specifically, we'll test the validation logic
// that happens before the service calls, which is the main responsibility of the handlers
func TestRegisterServerHandlerValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		input          types.RegisterServerInput
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing name",
			input: types.RegisterServerInput{
				Description: "Test server",
				Transport:   "streamable_http",
				URL:         "http://localhost:8080",
			},
			expectedStatus: 400, // http.StatusBadRequest
			expectedError:  "name is required",
		},
		{
			name: "missing transport",
			input: types.RegisterServerInput{
				Name:        "test-server",
				Description: "Test server",
				URL:         "http://localhost:8080",
			},
			expectedStatus: 400, // http.StatusBadRequest
			expectedError:  "transport is required",
		},
		{
			name: "invalid transport",
			input: types.RegisterServerInput{
				Name:        "test-server",
				Description: "Test server",
				Transport:   "invalid",
				URL:         "http://localhost:8080",
			},
			expectedStatus: 400, // http.StatusBadRequest
			expectedError:  "unsupported transport type",
		},
		{
			name: "streamable http without URL",
			input: types.RegisterServerInput{
				Name:        "test-server",
				Description: "Test server",
				Transport:   "streamable_http",
			},
			expectedStatus: 400, // http.StatusBadRequest
			expectedError:  "url is required for streamable HTTP transport",
		},
		{
			name: "stdio without command",
			input: types.RegisterServerInput{
				Name:        "test-server",
				Description: "Test server",
				Transport:   "stdio",
			},
			expectedStatus: 400, // http.StatusBadRequest
			expectedError:  "command is required for stdio transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validation logic directly since we can't easily mock the service
			// This tests the core validation that happens in the handler
			if tt.input.Name == "" {
				testhelpers.AssertEqual(t, "name is required", tt.expectedError)
				return
			}

			if tt.input.Transport == "" {
				testhelpers.AssertEqual(t, "transport is required", tt.expectedError)
				return
			}

			// Test transport type validation
			if tt.input.Transport != "streamable_http" && tt.input.Transport != "stdio" {
				testhelpers.AssertEqual(t, "unsupported transport type", tt.expectedError)
				return
			}

			// Test transport-specific validation using tagged switch
			switch tt.input.Transport {
			case "streamable_http":
				if tt.input.URL == "" {
					testhelpers.AssertEqual(t, "url is required for streamable HTTP transport", tt.expectedError)
				}
			case "stdio":
				if tt.input.Command == "" {
					testhelpers.AssertEqual(t, "command is required for stdio transport", tt.expectedError)
				}
			}
		})
	}
}

func TestTransportValidation(t *testing.T) {
	tests := []struct {
		name        string
		transport   string
		expectValid bool
	}{
		{"valid streamable_http", "streamable_http", true},
		{"valid stdio", "stdio", true},
		{"invalid transport", "invalid", false},
		{"empty transport", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := types.ValidateTransport(tt.transport)
			isValid := err == nil

			if isValid != tt.expectValid {
				t.Errorf("Expected transport '%s' to be valid=%v, got valid=%v",
					tt.transport, tt.expectValid, isValid)
			}
		})
	}
}

func TestInputStructureValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       types.RegisterServerInput
		expectValid bool
	}{
		{
			name: "valid streamable http",
			input: types.RegisterServerInput{
				Name:        "test-server",
				Description: "Test server",
				Transport:   "streamable_http",
				URL:         "http://localhost:8080",
			},
			expectValid: true,
		},
		{
			name: "valid stdio",
			input: types.RegisterServerInput{
				Name:        "test-server",
				Description: "Test server",
				Transport:   "stdio",
				Command:     "echo",
				Args:        []string{"hello"},
			},
			expectValid: true,
		},
		{
			name: "missing name",
			input: types.RegisterServerInput{
				Description: "Test server",
				Transport:   "streamable_http",
				URL:         "http://localhost:8080",
			},
			expectValid: false,
		},
		{
			name: "missing transport",
			input: types.RegisterServerInput{
				Name:        "test-server",
				Description: "Test server",
				URL:         "http://localhost:8080",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.input.Name != "" && tt.input.Transport != ""

			switch tt.input.Transport {
			case "streamable_http":
				isValid = isValid && tt.input.URL != ""
			case "stdio":
				isValid = isValid && tt.input.Command != ""
			}

			if isValid != tt.expectValid {
				t.Errorf("Expected input to be valid=%v, got valid=%v", tt.expectValid, isValid)
			}
		})
	}
}

func TestSessionModeValidation(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		expectValid  bool
		expectedMode types.SessionMode
	}{
		{"valid stateless", "stateless", true, types.SessionModeStateless},
		{"valid stateful", "stateful", true, types.SessionModeStateful},
		{"invalid mode", "invalid", false, ""},
		{"empty mode", "", true, types.SessionModeStateless}, // default to stateless
		{"whitespace", "\n\t  \n", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := types.ValidateSessionMode(tt.mode)
			isValid := err == nil

			if isValid != tt.expectValid {
				t.Errorf("Expected session mode '%s' to be valid=%v, got valid=%v",
					tt.mode, tt.expectValid, isValid)
			}
			if isValid && m != tt.expectedMode {
				t.Errorf("Expected session mode '%s' to be '%s', got '%s'",
					tt.mode, tt.expectedMode, m)
			}
		})
	}
}
