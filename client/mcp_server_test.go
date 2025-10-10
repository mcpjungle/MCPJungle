package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestRegisterServer(t *testing.T) {
	t.Parallel()

	t.Run("successful registration", func(t *testing.T) {
		expectedServer := &types.McpServer{
			Name:      "test-server",
			Transport: "stdio",
			Command:   "/usr/bin/test-server",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/servers") {
				t.Errorf("Expected path to end with /servers, got %s", r.URL.Path)
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

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(expectedServer)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		serverInput := &types.RegisterServerInput{
			Name:      "test-server",
			Transport: "stdio",
			Command:   "/usr/bin/test-server",
		}

		response, err := client.RegisterServer(serverInput)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if response.Name != expectedServer.Name {
			t.Errorf("Expected Name %s, got %s", expectedServer.Name, response.Name)
		}
		if response.Transport != expectedServer.Transport {
			t.Errorf("Expected Transport %s, got %s", expectedServer.Transport, response.Transport)
		}
		if response.Command != expectedServer.Command {
			t.Errorf("Expected Command %s, got %s", expectedServer.Command, response.Command)
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid server configuration"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		serverInput := &types.RegisterServerInput{
			Name:      "test-server",
			Transport: "stdio",
			Command:   "/usr/bin/test-server",
		}

		response, err := client.RegisterServer(serverInput)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if response != nil {
			t.Error("Expected nil response on error")
		}

		expectedError := "request failed with status: 400, message: Invalid server configuration"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})

	t.Run("network error", func(t *testing.T) {
		client := NewClient("http://invalid-url", "test-token", &http.Client{})
		serverInput := &types.RegisterServerInput{
			Name:      "test-server",
			Transport: "stdio",
			Command:   "/usr/bin/test-server",
		}

		response, err := client.RegisterServer(serverInput)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if response != nil {
			t.Error("Expected nil response on error")
		}

		if !strings.Contains(err.Error(), "failed to send request") {
			t.Errorf("Expected error to contain 'failed to send request', got %s", err.Error())
		}
	})
}

func TestListServers(t *testing.T) {
	t.Parallel()

	t.Run("successful list", func(t *testing.T) {
		expectedServers := []*types.McpServer{
			{
				Name:      "server1",
				Transport: "stdio",
				Command:   "/usr/bin/server1",
			},
			{
				Name:      "server2",
				Transport: "streamable_http",
				URL:       "http://localhost:8080",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/servers") {
				t.Errorf("Expected path to end with /servers, got %s", r.URL.Path)
			}

			// Verify authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-token" {
				t.Errorf("Expected Authorization header 'Bearer test-token', got %s", authHeader)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedServers)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		servers, err := client.ListServers()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(servers) != len(expectedServers) {
			t.Errorf("Expected %d servers, got %d", len(expectedServers), len(servers))
		}

		for i, server := range servers {
			if server.Name != expectedServers[i].Name {
				t.Errorf("Expected server[%d].Name %s, got %s", i, expectedServers[i].Name, server.Name)
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
		servers, err := client.ListServers()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(servers) != 0 {
			t.Errorf("Expected empty list, got %d servers", len(servers))
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		servers, err := client.ListServers()

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if servers != nil {
			t.Error("Expected nil servers on error")
		}

		expectedError := "request failed with status: 500, message: Internal server error"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestDeregisterServer(t *testing.T) {
	t.Parallel()

	t.Run("successful deregistration", func(t *testing.T) {
		serverName := "test-server"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != "DELETE" {
				t.Errorf("Expected DELETE method, got %s", r.Method)
			}
			expectedPath := "/api/v0/servers/" + serverName
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
		err := client.DeregisterServer(serverName)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("server not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Server not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		err := client.DeregisterServer("non-existent-server")

		if err == nil {
			t.Error("Expected error, got nil")
		}

		expectedError := "Server not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})

	t.Run("network error", func(t *testing.T) {
		client := NewClient("http://invalid-url", "test-token", &http.Client{})
		err := client.DeregisterServer("test-server")

		if err == nil {
			t.Error("Expected error, got nil")
		}

		if !strings.Contains(err.Error(), "failed to send request") {
			t.Errorf("Expected error to contain 'failed to send request', got %s", err.Error())
		}
	})
}
