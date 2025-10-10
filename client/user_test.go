package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestCreateUser(t *testing.T) {
	t.Parallel()

	t.Run("successful creation", func(t *testing.T) {
		expectedResponse := &types.CreateUserResponse{
			AccessToken: "user-access-token-12345",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/users") {
				t.Errorf("Expected path to end with /users, got %s", r.URL.Path)
			}

			// Verify content type
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Verify request body
			var createUserRequest types.CreateUserRequest
			if err := json.NewDecoder(r.Body).Decode(&createUserRequest); err != nil {
				t.Fatalf("Failed to decode request body: %v", err)
			}

			if createUserRequest.Username != "testuser" {
				t.Errorf("Expected Username 'testuser', got %s", createUserRequest.Username)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		createUserRequest := &types.CreateUserRequest{
			Username: "testuser",
		}

		response, err := client.CreateUser(createUserRequest)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if response.AccessToken != expectedResponse.AccessToken {
			t.Errorf("Expected AccessToken %s, got %s", expectedResponse.AccessToken, response.AccessToken)
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid user configuration"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		createUserRequest := &types.CreateUserRequest{
			Username: "testuser",
		}

		response, err := client.CreateUser(createUserRequest)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if response != nil {
			t.Error("Expected nil response on error")
		}

		expectedError := "request failed with status: 400, message: Invalid user configuration"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})

	t.Run("network error", func(t *testing.T) {
		client := NewClient("http://invalid-url", "test-token", &http.Client{})
		createUserRequest := &types.CreateUserRequest{
			Username: "testuser",
		}

		response, err := client.CreateUser(createUserRequest)

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

func TestListUsers(t *testing.T) {
	t.Parallel()

	t.Run("successful list", func(t *testing.T) {
		expectedUsers := []*types.User{
			{
				Username: "user1",
				Role:     "user",
			},
			{
				Username: "admin1",
				Role:     "admin",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET method, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/users") {
				t.Errorf("Expected path to end with /users, got %s", r.URL.Path)
			}

			// Verify authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-token" {
				t.Errorf("Expected Authorization header 'Bearer test-token', got %s", authHeader)
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedUsers)
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		users, err := client.ListUsers()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(users) != len(expectedUsers) {
			t.Errorf("Expected %d users, got %d", len(expectedUsers), len(users))
		}

		for i, user := range users {
			if user.Username != expectedUsers[i].Username {
				t.Errorf("Expected user[%d].Username %s, got %s", i, expectedUsers[i].Username, user.Username)
			}
			if user.Role != expectedUsers[i].Role {
				t.Errorf("Expected user[%d].Role %s, got %s", i, expectedUsers[i].Role, user.Role)
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
		users, err := client.ListUsers()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(users) != 0 {
			t.Errorf("Expected empty list, got %d users", len(users))
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		users, err := client.ListUsers()

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if users != nil {
			t.Error("Expected nil users on error")
		}

		expectedError := "request failed with status: 500, message: Internal server error"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()

	t.Run("successful deletion", func(t *testing.T) {
		username := "testuser"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != "DELETE" {
				t.Errorf("Expected DELETE method, got %s", r.Method)
			}
			expectedPath := "/api/v0/users/" + username
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
		err := client.DeleteUser(username)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("User not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL, "test-token", &http.Client{})
		err := client.DeleteUser("non-existent-user")

		if err == nil {
			t.Error("Expected error, got nil")
		}

		expectedError := "request failed with status: 404, message: User not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain %s, got %s", expectedError, err.Error())
		}
	})

	t.Run("network error", func(t *testing.T) {
		client := NewClient("http://invalid-url", "test-token", &http.Client{})
		err := client.DeleteUser("testuser")

		if err == nil {
			t.Error("Expected error, got nil")
		}

		if !strings.Contains(err.Error(), "failed to send request") {
			t.Errorf("Expected error to contain 'failed to send request', got %s", err.Error())
		}
	})
}

func TestCreateUserWithDifferentUsernames(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		username string
		expected string
	}{
		{"simple username", "testuser", "testuser"},
		{"username with numbers", "user123", "user123"},
		{"username with underscores", "test_user", "test_user"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request payload
				var createUserRequest types.CreateUserRequest
				if err := json.NewDecoder(r.Body).Decode(&createUserRequest); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				// Verify the username
				if createUserRequest.Username != tc.expected {
					t.Errorf("Expected Username %s, got %s", tc.expected, createUserRequest.Username)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				response := &types.CreateUserResponse{AccessToken: "test-token"}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token", &http.Client{})
			createUserRequest := &types.CreateUserRequest{
				Username: tc.username,
			}

			_, err := client.CreateUser(createUserRequest)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		})
	}
}
