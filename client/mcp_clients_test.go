package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestListMcpClients(t *testing.T) {
	t.Parallel()

	t.Run("successful list", func(t *testing.T) {
		expectedClients := []types.McpClient{
			{
				Name:        "client1",
				Description: "First test client",
				AllowList:   []string{"server1", "server2"},
			},
			{
				Name:        "client2",
				Description: "Second test client",
				AllowList:   []string{"server3"},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/clients") {
				t.Errorf("Expected path to end with /clients, got %s", r.URL.Path)
			}

			// Verify authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-token" {
				t.Errorf("Expected Authorization header 'Bearer test-token', got %s", authHeader)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedClients)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		clients, err := client.ListMcpClients()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(clients) != len(expectedClients) {
			t.Errorf("Expected %d clients, got %d", len(expectedClients), len(clients))
		}

		for i, client := range clients {
			if client.Name != expectedClients[i].Name {
				t.Errorf("Expected client[%d].Name %s, got %s", i, expectedClients[i].Name, client.Name)
			}
			if client.Description != expectedClients[i].Description {
				t.Errorf("Expected client[%d].Description %s, got %s", i, expectedClients[i].Description, client.Description)
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
		clients, err := client.ListMcpClients()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(clients) != 0 {
			t.Errorf("Expected empty list, got %d clients", len(clients))
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		clients, err := client.ListMcpClients()

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if clients != nil {
			t.Error("Expected nil clients on error")
		}

		expectedError := "request failed with status: 500, message: Internal server error"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})

	t.Run("network error", func(t *testing.T) {
		client := NewClient("http://invalid-url", "test-token", &http.Client{})
		clients, err := client.ListMcpClients()

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if clients != nil {
			t.Error("Expected nil clients on error")
		}

		if !strings.Contains(err.Error(), "failed to send request") {
			t.Errorf("Expected error to contain 'failed to send request', got %s", err.Error())
		}
	})
}

func TestDeleteMcpClient(t *testing.T) {
	t.Parallel()

	t.Run("successful deletion", func(t *testing.T) {
		clientName := "test-client"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != "DELETE" {
				t.Errorf("Expected DELETE method, got %s", r.Method)
			}
			expectedPath := "/api/v0/clients/" + clientName
			if !strings.HasSuffix(r.URL.Path, expectedPath) {
				t.Errorf("Expected path to end with %s, got %s", expectedPath, r.URL.Path)
			}

			// Verify authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-token" {
				t.Errorf("Expected Authorization header 'Bearer test-token', got %s", authHeader)
			}

			// Return success response
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		err := client.DeleteMcpClient(clientName)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("client not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Client not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		err := client.DeleteMcpClient("non-existent-client")

		if err == nil {
			t.Error("Expected error, got nil")
		}

		expectedError := "request failed with status: 404, message: Client not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})

	t.Run("network error", func(t *testing.T) {
		client := NewClient("http://invalid-url", "test-token", &http.Client{})
		err := client.DeleteMcpClient("test-client")

		if err == nil {
			t.Error("Expected error, got nil")
		}

		if !strings.Contains(err.Error(), "failed to send request") {
			t.Errorf("Expected error to contain 'failed to send request', got %s", err.Error())
		}
	})
}

func TestCreateMcpClient(t *testing.T) {
	t.Parallel()

	t.Run("successful creation", func(t *testing.T) {
		expectedToken := "client-access-token-12345"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/clients") {
				t.Errorf("Expected path to end with /clients, got %s", r.URL.Path)
			}

			// Verify content type
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Verify authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-token" {
				t.Errorf("Expected Authorization header 'Bearer test-token', got %s", authHeader)
			}

			// Verify request body
			var mcpClient types.McpClient
			if err := json.NewDecoder(r.Body).Decode(&mcpClient); err != nil {
				t.Fatalf("Failed to decode request body: %v", err)
			}

			if mcpClient.Name != "test-client" {
				t.Errorf("Expected Name 'test-client', got %s", mcpClient.Name)
			}
			if mcpClient.Description != "Test client description" {
				t.Errorf("Expected Description 'Test client description', got %s", mcpClient.Description)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			response := struct {
				AccessToken string `json:"access_token"`
			}{
				AccessToken: expectedToken,
			}
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		mcpClient := &types.McpClient{
			Name:        "test-client",
			Description: "Test client description",
			AllowList:   []string{"server1", "server2"},
		}

		accessToken, err := client.CreateMcpClient(mcpClient)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if accessToken != expectedToken {
			t.Errorf("Expected access token %s, got %s", expectedToken, accessToken)
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid client configuration"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		mcpClient := &types.McpClient{
			Name:        "test-client",
			Description: "Test client description",
			AllowList:   []string{"server1"},
		}

		accessToken, err := client.CreateMcpClient(mcpClient)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if accessToken != "" {
			t.Error("Expected empty access token on error")
		}

		expectedError := "request failed with status: 400, message: Invalid client configuration"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})

	t.Run("network error", func(t *testing.T) {
		client := NewClient("http://invalid-url", "test-token", &http.Client{})
		mcpClient := &types.McpClient{
			Name:        "test-client",
			Description: "Test client description",
			AllowList:   []string{"server1"},
		}

		accessToken, err := client.CreateMcpClient(mcpClient)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if accessToken != "" {
			t.Error("Expected empty access token on error")
		}

		if !strings.Contains(err.Error(), "failed to send request") {
			t.Errorf("Expected error to contain 'failed to send request', got %s", err.Error())
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		mcpClient := &types.McpClient{
			Name:        "test-client",
			Description: "Test client description",
			AllowList:   []string{"server1"},
		}

		accessToken, err := client.CreateMcpClient(mcpClient)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if accessToken != "" {
			t.Error("Expected empty access token on error")
		}

		if !strings.Contains(err.Error(), "failed to decode response") {
			t.Errorf("Expected error to contain 'failed to decode response', got %s", err.Error())
		}
	})
}

func TestCreateMcpClientWithEmptyAllowList(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request payload
		var mcpClient types.McpClient
		if err := json.NewDecoder(r.Body).Decode(&mcpClient); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify empty allow list
		if len(mcpClient.AllowList) != 0 {
			t.Errorf("Expected empty AllowList, got %v", mcpClient.AllowList)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		response := struct {
			AccessToken string `json:"access_token"`
		}{
			AccessToken: "test-token",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", &http.Client{})
	mcpClient := &types.McpClient{
		Name:        "test-client",
		Description: "Test client with empty allow list",
		AllowList:   []string{},
	}

	_, err := client.CreateMcpClient(mcpClient)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}
