package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestListResources(t *testing.T) {
	t.Parallel()

	t.Run("successful list without filter", func(t *testing.T) {
		expectedResources := []*types.Resource{
			{
				URI:         "mcpj://res/polaro/c3lzdGVtOi8vYmF0dGVyeQ",
				Name:        "polaro__system_battery",
				Enabled:     true,
				Description: "Battery Status",
				MIMEType:    "application/json",
			},
			{
				URI:         "mcpj://res/polaro/c3lzdGVtOi8vbmV0d29yaw",
				Name:        "polaro__system_network",
				Enabled:     false,
				Description: "Network Status",
				MIMEType:    "application/json",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/resources") {
				t.Errorf("Expected path to end with /resources, got %s", r.URL.Path)
			}
			if r.URL.RawQuery != "" {
				t.Errorf("Expected no query parameters, got %s", r.URL.RawQuery)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedResources)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		resources, err := client.ListResources("")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(resources) != len(expectedResources) {
			t.Errorf("Expected %d resources, got %d", len(expectedResources), len(resources))
		}
		for i, resource := range resources {
			if resource.URI != expectedResources[i].URI {
				t.Errorf("Expected resource[%d].URI %s, got %s", i, expectedResources[i].URI, resource.URI)
			}
		}
	})

	t.Run("successful list with server filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serverParam := r.URL.Query().Get("server")
			if serverParam != "polaro" {
				t.Errorf("Expected server query param 'polaro', got %s", serverParam)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]*types.Resource{})
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		_, err := client.ListResources("polaro")
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
		resources, err := client.ListResources("")

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if resources != nil {
			t.Error("Expected nil resources on error")
		}

		expectedError := "request failed with status: 500, message: Internal server error"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestGetResource(t *testing.T) {
	t.Parallel()

	expected := &types.Resource{
		URI:         "mcpj://res/polaro/c3lzdGVtOi8vc3lzdGVtL2luZm8",
		Name:        "polaro__system_info",
		Enabled:     true,
		Description: "CPU load, memory, disk, and uptime",
		MIMEType:    "application/json",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/resources/get") {
			t.Errorf("Expected path to end with /resources/get, got %s", r.URL.Path)
		}
		var request types.ResourceGetRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if request.URI != expected.URI {
			t.Errorf("Expected uri=%s, got %s", expected.URI, request.URI)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", &http.Client{})
	resource, err := client.GetResource(expected.URI)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resource.URI != expected.URI {
		t.Fatalf("Expected URI %s, got %s", expected.URI, resource.URI)
	}
}

func TestReadResource(t *testing.T) {
	t.Parallel()

	expected := &types.ResourceReadResult{
		Contents: []map[string]any{
			{
				"uri":      "mcpj://res/polaro/c3lzdGVtOi8vc3lzdGVtL2luZm8",
				"mimeType": "application/json",
				"text":     "{\"uptime\":\"up 1 hour\"}",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/resources/read") {
			t.Errorf("Expected path to end with /resources/read, got %s", r.URL.Path)
		}

		var request types.ResourceReadRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if request.URI != "mcpj://res/polaro/c3lzdGVtOi8vc3lzdGVtL2luZm8" {
			t.Errorf("Expected MCPJungle URI, got %s", request.URI)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", &http.Client{})
	result, err := client.ReadResource("mcpj://res/polaro/c3lzdGVtOi8vc3lzdGVtL2luZm8")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Contents))
	}
}
