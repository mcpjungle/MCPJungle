package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestCreateToolGroup(t *testing.T) {
	t.Parallel()

	t.Run("successful creation", func(t *testing.T) {
		expectedResponse := &types.CreateToolGroupResponse{
			ToolGroupEndpoints: &types.ToolGroupEndpoints{
				StreamableHTTPEndpoint: "/api/v0/tool-groups/test-group",
				SSEEndpoint:            "/api/v0/tool-groups/test-group/sse",
				SSEMessageEndpoint:     "/api/v0/tool-groups/test-group/sse/message",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tool-groups") {
				t.Errorf("Expected path to end with /tool-groups, got %s", r.URL.Path)
			}

			// Verify content type
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Verify request body
			var toolGroup types.ToolGroup
			if err := json.NewDecoder(r.Body).Decode(&toolGroup); err != nil {
				t.Fatalf("Failed to decode request body: %v", err)
			}

			if toolGroup.Name != "test-group" {
				t.Errorf("Expected Name 'test-group', got %s", toolGroup.Name)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		toolGroup := &types.ToolGroup{
			Name:          "test-group",
			Description:   "Test tool group",
			IncludedTools: []string{"tool1", "tool2"},
		}

		response, err := client.CreateToolGroup(toolGroup)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if response.StreamableHTTPEndpoint != expectedResponse.StreamableHTTPEndpoint {
			t.Errorf("Expected StreamableHTTPEndpoint %s, got %s", expectedResponse.StreamableHTTPEndpoint, response.StreamableHTTPEndpoint)
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid tool group configuration"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		toolGroup := &types.ToolGroup{
			Name:          "test-group",
			Description:   "Test tool group",
			IncludedTools: []string{"tool1"},
		}

		response, err := client.CreateToolGroup(toolGroup)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if response != nil {
			t.Error("Expected nil response on error")
		}

		expectedError := "request failed with status: 400, message: Invalid tool group configuration"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestGetToolGroup(t *testing.T) {
	t.Parallel()

	t.Run("successful retrieval", func(t *testing.T) {
		expectedGroup := &types.ToolGroup{
			Name:          "test-group",
			Description:   "Test tool group",
			IncludedTools: []string{"tool1", "tool2"},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tool-groups/test-group") {
				t.Errorf("Expected path to end with /tool-groups/test-group, got %s", r.URL.Path)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedGroup)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		group, err := client.GetToolGroup("test-group")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if group.Name != expectedGroup.Name {
			t.Errorf("Expected Name %s, got %s", expectedGroup.Name, group.Name)
		}
		if group.Description != expectedGroup.Description {
			t.Errorf("Expected Description %s, got %s", expectedGroup.Description, group.Description)
		}
		if len(group.IncludedTools) != len(expectedGroup.IncludedTools) {
			t.Errorf("Expected IncludedTools length %d, got %d", len(expectedGroup.IncludedTools), len(group.IncludedTools))
		}
	})

	t.Run("group not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Tool group not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		group, err := client.GetToolGroup("non-existent-group")

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if group != nil {
			t.Error("Expected nil group on error")
		}

		expectedError := "request failed with status: 404, message: Tool group not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestDeleteToolGroup(t *testing.T) {
	t.Parallel()

	t.Run("successful deletion", func(t *testing.T) {
		groupName := "test-group"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != "DELETE" {
				t.Errorf("Expected DELETE method, got %s", r.Method)
			}
			expectedPath := "/api/v0/tool-groups/" + groupName
			if !strings.HasSuffix(r.URL.Path, expectedPath) {
				t.Errorf("Expected path to end with %s, got %s", expectedPath, r.URL.Path)
			}

			// Return success response
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		err := client.DeleteToolGroup(groupName)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("group not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Tool group not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		err := client.DeleteToolGroup("non-existent-group")

		if err == nil {
			t.Error("Expected error, got nil")
		}

		expectedError := "request failed with status: 404, message: Tool group not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestListToolGroups(t *testing.T) {
	t.Parallel()

	t.Run("successful list", func(t *testing.T) {
		expectedGroups := []*types.ToolGroup{
			{
				Name:          "group1",
				Description:   "First group",
				IncludedTools: []string{"tool1"},
			},
			{
				Name:          "group2",
				Description:   "Second group",
				IncludedTools: []string{"tool2", "tool3"},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tool-groups") {
				t.Errorf("Expected path to end with /tool-groups, got %s", r.URL.Path)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedGroups)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		groups, err := client.ListToolGroups()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(groups) != len(expectedGroups) {
			t.Errorf("Expected %d groups, got %d", len(expectedGroups), len(groups))
		}

		for i, group := range groups {
			if group.Name != expectedGroups[i].Name {
				t.Errorf("Expected group[%d].Name %s, got %s", i, expectedGroups[i].Name, group.Name)
			}
		}
	})

	t.Run("empty list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[]"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		groups, err := client.ListToolGroups()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(groups) != 0 {
			t.Errorf("Expected empty list, got %d groups", len(groups))
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		groups, err := client.ListToolGroups()

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if groups != nil {
			t.Error("Expected nil groups on error")
		}

		expectedError := "request failed with status: 500, message: Internal server error"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestGetToolGroupEffectiveTools(t *testing.T) {
	t.Parallel()

	t.Run("successful retrieval", func(t *testing.T) {
		expectedTools := []string{"alpha", "beta"}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/tool-groups/test-group/effective-tools") {
				t.Errorf("Expected path to end with /tool-groups/test-group/effective-tools, got %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"tools": expectedTools})
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tools, err := client.GetToolGroupEffectiveTools("test-group")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(tools) != len(expectedTools) {
			t.Fatalf("Expected %d tools, got %d", len(expectedTools), len(tools))
		}
		for i := range expectedTools {
			if tools[i] != expectedTools[i] {
				t.Errorf("Expected tools[%d] to be %s, got %s", i, expectedTools[i], tools[i])
			}
		}
	})

	t.Run("group not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Tool group not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		tools, err := client.GetToolGroupEffectiveTools("missing")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if tools != nil {
			t.Fatal("Expected nil tools on error")
		}
	})
}
