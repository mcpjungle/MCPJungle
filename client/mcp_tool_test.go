package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestListTools(t *testing.T) {
	t.Parallel()

	t.Run("successful list without filter", func(t *testing.T) {
		expectedTools := []*types.Tool{
			{
				Name:        "tool1",
				Enabled:     true,
				Description: "First tool",
			},
			{
				Name:        "tool2",
				Enabled:     false,
				Description: "Second tool",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tools") {
				t.Errorf("Expected path to end with /tools, got %s", r.URL.Path)
			}

			// Verify no query parameters
			if r.URL.RawQuery != "" {
				t.Errorf("Expected no query parameters, got %s", r.URL.RawQuery)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedTools)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tools, err := client.ListTools("")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(tools) != len(expectedTools) {
			t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
		}

		for i, tool := range tools {
			if tool.Name != expectedTools[i].Name {
				t.Errorf("Expected tool[%d].Name %s, got %s", i, expectedTools[i].Name, tool.Name)
			}
		}
	})

	t.Run("successful list with server filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify query parameter
			serverParam := r.URL.Query().Get("server")
			if serverParam != "test-server" {
				t.Errorf("Expected server query param 'test-server', got %s", serverParam)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]*types.Tool{})
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		_, err := client.ListTools("test-server")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tools, err := client.ListTools("")

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if tools != nil {
			t.Error("Expected nil tools on error")
		}

		expectedError := "request failed with status: 500, message: Internal server error"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestEnableTools(t *testing.T) {
	t.Parallel()

	t.Run("successful enable", func(t *testing.T) {
		expectedTools := []string{"tool1", "tool2"}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tools/enable") {
				t.Errorf("Expected path to end with /tools/enable, got %s", r.URL.Path)
			}

			// Verify query parameter
			entityParam := r.URL.Query().Get("entity")
			if entityParam != "test-tool" {
				t.Errorf("Expected entity query param 'test-tool', got %s", entityParam)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedTools)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tools, err := client.EnableTools("test-tool")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(tools) != len(expectedTools) {
			t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
		}

		for i, tool := range tools {
			if tool != expectedTools[i] {
				t.Errorf("Expected tool[%d] %s, got %s", i, expectedTools[i], tool)
			}
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Tool not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tools, err := client.EnableTools("non-existent-tool")

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if tools != nil {
			t.Error("Expected nil tools on error")
		}

		expectedError := "request failed with status: 400, message: Tool not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestDisableTools(t *testing.T) {
	t.Parallel()

	t.Run("successful disable", func(t *testing.T) {
		expectedTools := []string{"tool1", "tool2"}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tools/disable") {
				t.Errorf("Expected path to end with /tools/disable, got %s", r.URL.Path)
			}

			// Verify query parameter
			entityParam := r.URL.Query().Get("entity")
			if entityParam != "test-tool" {
				t.Errorf("Expected entity query param 'test-tool', got %s", entityParam)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedTools)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tools, err := client.DisableTools("test-tool")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(tools) != len(expectedTools) {
			t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Tool not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tools, err := client.DisableTools("non-existent-tool")

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if tools != nil {
			t.Error("Expected nil tools on error")
		}

		expectedError := "request failed with status: 400, message: Tool not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestInvokeTool(t *testing.T) {
	t.Parallel()

	t.Run("successful invocation", func(t *testing.T) {
		expectedResult := &types.ToolInvokeResult{
			Meta:    map[string]any{"status": "success"},
			IsError: false,
			Content: []map[string]any{{"result": "successful"}},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tools/invoke") {
				t.Errorf("Expected path to end with /tools/invoke, got %s", r.URL.Path)
			}

			// Verify content type
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Verify request body
			var requestBody map[string]any
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("Failed to decode request body: %v", err)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedResult)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		input := map[string]any{"param": "value"}
		result, err := client.InvokeTool("test-tool", input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.IsError != expectedResult.IsError {
			t.Errorf("Expected IsError %v, got %v", expectedResult.IsError, result.IsError)
		}
		if len(result.Content) != len(expectedResult.Content) {
			t.Errorf("Expected Content length %d, got %d", len(expectedResult.Content), len(result.Content))
		}
	})

	t.Run("tool error response", func(t *testing.T) {
		expectedResult := &types.ToolInvokeResult{
			Meta:    map[string]any{"error": "tool execution failed"},
			IsError: true,
			Content: []map[string]any{{"error": "Invalid input"}},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedResult)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		input := map[string]any{"invalid": "input"}
		result, err := client.InvokeTool("test-tool", input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("Expected IsError to be true")
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Tool not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		input := map[string]any{"param": "value"}
		result, err := client.InvokeTool("non-existent-tool", input)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if result != nil {
			t.Error("Expected nil result on error")
		}

		expectedError := "request failed with status: 400, message: Tool not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestGetTool(t *testing.T) {
	t.Parallel()

	t.Run("successful tool retrieval", func(t *testing.T) {
		expectedTool := &types.Tool{
			Name:        "test-tool",
			Enabled:     true,
			Description: "Test tool description",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tool") {
				t.Errorf("Expected path to end with /tool, got %s", r.URL.Path)
			}

			// Verify query parameter
			nameParam := r.URL.Query().Get("name")
			if nameParam != "test-tool" {
				t.Errorf("Expected name query param 'test-tool', got %s", nameParam)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedTool)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tool, err := client.GetTool("test-tool")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if tool.Name != expectedTool.Name {
			t.Errorf("Expected Name %s, got %s", expectedTool.Name, tool.Name)
		}
		if tool.Enabled != expectedTool.Enabled {
			t.Errorf("Expected Enabled %v, got %v", expectedTool.Enabled, tool.Enabled)
		}
	})

	t.Run("tool not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Tool not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tool, err := client.GetTool("non-existent-tool")

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if tool != nil {
			t.Error("Expected nil tool on error")
		}

		expectedError := "request failed with status: 404, message: Tool not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}
