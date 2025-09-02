package types

import (
	"encoding/json"
	"testing"
)

func TestUserRole(t *testing.T) {
	t.Run("UserRole constants", func(t *testing.T) {
		if UserRoleAdmin != "admin" {
			t.Errorf("Expected UserRoleAdmin to be 'admin', got %s", UserRoleAdmin)
		}
		if UserRoleUser != "user" {
			t.Errorf("Expected UserRoleUser to be 'user', got %s", UserRoleUser)
		}
	})

	t.Run("UserRole string conversion", func(t *testing.T) {
		adminRole := string(UserRoleAdmin)
		userRole := string(UserRoleUser)

		if adminRole != "admin" {
			t.Errorf("Expected adminRole string to be 'admin', got %s", adminRole)
		}
		if userRole != "user" {
			t.Errorf("Expected userRole string to be 'user', got %s", userRole)
		}
	})
}

func TestUser(t *testing.T) {
	t.Run("User struct creation", func(t *testing.T) {
		user := User{
			Username: "testuser",
			Role:     "user",
		}

		if user.Username != "testuser" {
			t.Errorf("Expected Username to be 'testuser', got %s", user.Username)
		}
		if user.Role != "user" {
			t.Errorf("Expected Role to be 'user', got %s", user.Role)
		}
	})

	t.Run("User struct zero values", func(t *testing.T) {
		var user User

		if user.Username != "" {
			t.Errorf("Expected empty Username, got %s", user.Username)
		}
		if user.Role != "" {
			t.Errorf("Expected empty Role, got %s", user.Role)
		}
	})

	t.Run("User JSON marshaling", func(t *testing.T) {
		user := User{
			Username: "testuser",
			Role:     "admin",
		}

		data, err := json.Marshal(user)
		if err != nil {
			t.Fatalf("Failed to marshal User: %v", err)
		}

		expected := `{"username":"testuser","role":"admin"}`
		if string(data) != expected {
			t.Errorf("Expected JSON %s, got %s", expected, string(data))
		}
	})

	t.Run("User JSON unmarshaling", func(t *testing.T) {
		jsonData := `{"username":"testuser","role":"user"}`
		var user User

		err := json.Unmarshal([]byte(jsonData), &user)
		if err != nil {
			t.Fatalf("Failed to unmarshal User: %v", err)
		}

		if user.Username != "testuser" {
			t.Errorf("Expected Username 'testuser', got %s", user.Username)
		}
		if user.Role != "user" {
			t.Errorf("Expected Role 'user', got %s", user.Role)
		}
	})
}

func TestCreateUserRequest(t *testing.T) {
	t.Run("CreateUserRequest struct creation", func(t *testing.T) {
		req := CreateUserRequest{
			Username: "newuser",
		}

		if req.Username != "newuser" {
			t.Errorf("Expected Username to be 'newuser', got %s", req.Username)
		}
	})

	t.Run("CreateUserRequest struct zero values", func(t *testing.T) {
		var req CreateUserRequest

		if req.Username != "" {
			t.Errorf("Expected empty Username, got %s", req.Username)
		}
	})

	t.Run("CreateUserRequest JSON marshaling", func(t *testing.T) {
		req := CreateUserRequest{
			Username: "newuser",
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal CreateUserRequest: %v", err)
		}

		expected := `{"username":"newuser"}`
		if string(data) != expected {
			t.Errorf("Expected JSON %s, got %s", expected, string(data))
		}
	})

	t.Run("CreateUserRequest JSON unmarshaling", func(t *testing.T) {
		jsonData := `{"username":"newuser"}`
		var req CreateUserRequest

		err := json.Unmarshal([]byte(jsonData), &req)
		if err != nil {
			t.Fatalf("Failed to unmarshal CreateUserRequest: %v", err)
		}

		if req.Username != "newuser" {
			t.Errorf("Expected Username 'newuser', got %s", req.Username)
		}
	})
}

func TestCreateUserResponse(t *testing.T) {
	t.Run("CreateUserResponse struct creation", func(t *testing.T) {
		resp := CreateUserResponse{
			Username:    "newuser",
			Role:        "user",
			AccessToken: "token123",
		}

		if resp.Username != "newuser" {
			t.Errorf("Expected Username to be 'newuser', got %s", resp.Username)
		}
		if resp.Role != "user" {
			t.Errorf("Expected Role to be 'user', got %s", resp.Role)
		}
		if resp.AccessToken != "token123" {
			t.Errorf("Expected AccessToken to be 'token123', got %s", resp.AccessToken)
		}
	})

	t.Run("CreateUserResponse struct zero values", func(t *testing.T) {
		var resp CreateUserResponse

		if resp.Username != "" {
			t.Errorf("Expected empty Username, got %s", resp.Username)
		}
		if resp.Role != "" {
			t.Errorf("Expected empty Role, got %s", resp.Role)
		}
		if resp.AccessToken != "" {
			t.Errorf("Expected empty AccessToken, got %s", resp.AccessToken)
		}
	})

	t.Run("CreateUserResponse JSON marshaling", func(t *testing.T) {
		resp := CreateUserResponse{
			Username:    "newuser",
			Role:        "admin",
			AccessToken: "admin_token_456",
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Failed to marshal CreateUserResponse: %v", err)
		}

		expected := `{"username":"newuser","role":"admin","access_token":"admin_token_456"}`
		if string(data) != expected {
			t.Errorf("Expected JSON %s, got %s", expected, string(data))
		}
	})

	t.Run("CreateUserResponse JSON unmarshaling", func(t *testing.T) {
		jsonData := `{"username":"newuser","role":"user","access_token":"user_token_789"}`
		var resp CreateUserResponse

		err := json.Unmarshal([]byte(jsonData), &resp)
		if err != nil {
			t.Fatalf("Failed to unmarshal CreateUserResponse: %v", err)
		}

		if resp.Username != "newuser" {
			t.Errorf("Expected Username 'newuser', got %s", resp.Username)
		}
		if resp.Role != "user" {
			t.Errorf("Expected Role 'user', got %s", resp.Role)
		}
		if resp.AccessToken != "user_token_789" {
			t.Errorf("Expected AccessToken 'user_token_789', got %s", resp.AccessToken)
		}
	})
}

func TestUserTypesIntegration(t *testing.T) {
	t.Run("User role validation", func(t *testing.T) {
		validRoles := []string{string(UserRoleAdmin), string(UserRoleUser)}
		user := User{
			Role: string(UserRoleUser),
		}

		found := false
		for _, role := range validRoles {
			if user.Role == role {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("User role '%s' is not in valid roles list: %v", user.Role, validRoles)
		}
	})

	t.Run("CreateUserRequest to CreateUserResponse mapping", func(t *testing.T) {
		req := CreateUserRequest{
			Username: "testuser",
		}

		resp := CreateUserResponse{
			Username:    req.Username,
			Role:        string(UserRoleUser),
			AccessToken: "generated_token",
		}

		if resp.Username != req.Username {
			t.Errorf("Response Username should match request Username")
		}
		if resp.Role != string(UserRoleUser) {
			t.Errorf("Response Role should be UserRoleUser")
		}
		if resp.AccessToken == "" {
			t.Error("Response AccessToken should not be empty")
		}
	})
}
