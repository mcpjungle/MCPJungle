package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	baseURL := "https://api.example.com"
	accessToken := "test-token"
	httpClient := &http.Client{}

	client := NewClient(baseURL, accessToken, httpClient)

	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL %s, got %s", baseURL, client.baseURL)
	}
	if client.accessToken != accessToken {
		t.Errorf("Expected accessToken %s, got %s", accessToken, client.accessToken)
	}
	if client.httpClient != httpClient {
		t.Error("Expected httpClient to match provided client")
	}
}

func TestNewClientWithEmptyToken(t *testing.T) {
	t.Parallel()

	client := NewClient("https://api.example.com", "", &http.Client{})

	if client.accessToken != "" {
		t.Errorf("Expected empty accessToken, got %s", client.accessToken)
	}
}

func TestConstructAPIEndpoint(t *testing.T) {
	t.Parallel()

	client := NewClient("https://api.example.com", "token", &http.Client{})

	tests := []struct {
		name         string
		suffixPath   string
		expectedPath string
	}{
		{
			name:         "simple path",
			suffixPath:   "servers",
			expectedPath: "https://api.example.com/api/v0/servers",
		},
		{
			name:         "nested path",
			suffixPath:   "servers/test-server",
			expectedPath: "https://api.example.com/api/v0/servers/test-server",
		},
		{
			name:         "empty suffix",
			suffixPath:   "",
			expectedPath: "https://api.example.com/api/v0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.constructAPIEndpoint(tt.suffixPath)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expectedPath {
				t.Errorf("Expected %s, got %s", tt.expectedPath, result)
			}
		})
	}
}

func TestNewRequest(t *testing.T) {
	t.Parallel()

	client := NewClient("https://api.example.com", "test-token", &http.Client{})

	t.Run("request with access token", func(t *testing.T) {
		body := strings.NewReader("test body")
		req, err := client.newRequest(http.MethodPost, "https://api.example.com/test", body)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if req.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", req.Method)
		}

		if req.URL.String() != "https://api.example.com/test" {
			t.Errorf("Expected URL https://api.example.com/test, got %s", req.URL.String())
		}

		authHeader := req.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got %s", authHeader)
		}
	})

	t.Run("request without access token", func(t *testing.T) {
		clientNoToken := NewClient("https://api.example.com", "", &http.Client{})
		req, err := clientNoToken.newRequest(http.MethodGet, "https://api.example.com/test", nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		authHeader := req.Header.Get("Authorization")
		if authHeader != "" {
			t.Errorf("Expected empty Authorization header, got %s", authHeader)
		}
	})

	t.Run("request with nil body", func(t *testing.T) {
		req, err := client.newRequest(http.MethodGet, "https://api.example.com/test", nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if req.Body != nil {
			t.Error("Expected nil body, got non-nil")
		}
	})
}

func TestNewRequestWithInvalidURL(t *testing.T) {
	t.Parallel()

	client := NewClient("https://api.example.com", "token", &http.Client{})

	// Test with invalid URL
	req, err := client.newRequest(http.MethodGet, "://invalid-url", nil)
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
	if req != nil {
		t.Error("Expected nil request for invalid URL")
	}
}

func TestClientIntegration(t *testing.T) {
	t.Parallel()

	// Test that client can make actual HTTP requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got %s", authHeader)
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", &http.Client{})

	// Test constructAPIEndpoint + newRequest integration
	endpoint, err := client.constructAPIEndpoint("test")
	if err != nil {
		t.Fatalf("Failed to construct endpoint: %v", err)
	}

	req, err := client.newRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Make the actual request
	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expectedBody := `{"message": "success"}`
	if string(body) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, string(body))
	}
}

func TestClientWithDifferentHTTPClients(t *testing.T) {
	t.Parallel()

	baseURL := "https://api.example.com"
	accessToken := "test-token"

	// Test with default HTTP client
	client1 := NewClient(baseURL, accessToken, &http.Client{})
	if client1.httpClient == nil {
		t.Error("Expected non-nil httpClient")
	}

	// Test with custom HTTP client
	customClient := &http.Client{
		Timeout: 30, // This would be a proper timeout in real usage
	}
	client2 := NewClient(baseURL, accessToken, customClient)
	if client2.httpClient != customClient {
		t.Error("Expected httpClient to match custom client")
	}
}

func TestConstructAPIEndpointWithComplexPaths(t *testing.T) {
	t.Parallel()

	client := NewClient("https://api.example.com", "token", &http.Client{})

	tests := []struct {
		name         string
		suffixPath   string
		expectedPath string
	}{
		{
			name:         "path with query params",
			suffixPath:   "tools?server=test",
			expectedPath: "https://api.example.com/api/v0/tools%3Fserver=test",
		},
		{
			name:         "path with special characters",
			suffixPath:   "servers/test-server-123",
			expectedPath: "https://api.example.com/api/v0/servers/test-server-123",
		},
		{
			name:         "path with multiple segments",
			suffixPath:   "tool-groups/my-group/tools",
			expectedPath: "https://api.example.com/api/v0/tool-groups/my-group/tools",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.constructAPIEndpoint(tt.suffixPath)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expectedPath {
				t.Errorf("Expected %s, got %s", tt.expectedPath, result)
			}
		})
	}
}

// badReader simulates a Read error
type badReader struct{}

func (badReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (badReader) Close() error             { return nil }

func TestParseErrorResponse(t *testing.T) {
	t.Parallel()

	client := NewClient("https://api.example.com", "token", &http.Client{})

	tests := []struct {
		name           string
		statusCode     int
		body           string
		expectContains string
		expectErr      bool
		readErr        bool
	}{
		{
			name:           "valid JSON error",
			statusCode:     400,
			body:           `{"error":"something went wrong"}`,
			expectContains: "something went wrong",
			expectErr:      true,
		},
		{
			name:           "valid JSON error but empty message",
			statusCode:     404,
			body:           `{"error":""}`,
			expectContains: "request failed with status: 404",
			expectErr:      true,
		},
		{
			name:           "invalid JSON",
			statusCode:     500,
			body:           `not a json`,
			expectContains: "request failed with status: 500, message: not a json",
			expectErr:      true,
		},
		{
			name:           "non-JSON, non-error status",
			statusCode:     200,
			body:           `all good`,
			expectContains: "unexpected response with status: 200, body: all good",
			expectErr:      true,
		},
		{
			name:           "valid JSON, non-error status",
			statusCode:     201,
			body:           `{"error":"should not parse"}`,
			expectContains: "unexpected response with status: 201, body: {\"error\":\"should not parse\"}",
			expectErr:      true,
		},
		{
			name:           "read error from body",
			statusCode:     500,
			body:           "",
			expectContains: "request failed with status: 500 (unable to read error details)",
			expectErr:      true,
			readErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.ReadCloser
			if tt.readErr {
				// Simulate read error
				body = io.NopCloser(badReader{})
			} else {
				body = io.NopCloser(strings.NewReader(tt.body))
			}
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       body,
			}
			err := client.parseErrorResponse(resp)
			if tt.expectErr && err == nil {
				t.Fatalf("Expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("Did not expect error, got %v", err)
			}
			if err != nil && !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("Expected error to contain %q, got %q", tt.expectContains, err.Error())
			}
		})
	}
}
