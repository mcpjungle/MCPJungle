package types

import (
	"encoding/json"
	"testing"
)

func TestUserRole(t *testing.T) {
	t.Parallel()

	// Test UserRole constants
	if UserRoleAdmin != "admin" {
		t.Errorf("Expected UserRoleAdmin to be 'admin', got %s", UserRoleAdmin)
	}
	if UserRoleUser != "user" {
		t.Errorf("Expected UserRoleUser to be 'user', got %s", UserRoleUser)
	}

	// Test string conversion
	adminRole := string(UserRoleAdmin)
	userRole := string(UserRoleUser)

	if adminRole != "admin" {
		t.Errorf("Expected adminRole string to be 'admin', got %s", adminRole)
	}
	if userRole != "user" {
		t.Errorf("Expected userRole string to be 'user', got %s", userRole)
	}
}

func TestUser(t *testing.T) {
	t.Parallel()

	// Test struct creation
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
}

func TestUserZeroValues(t *testing.T) {
	t.Parallel()

	var user User

	if user.Username != "" {
		t.Errorf("Expected empty Username, got %s", user.Username)
	}
	if user.Role != "" {
		t.Errorf("Expected empty Role, got %s", user.Role)
	}
}

func TestUserJSONMarshaling(t *testing.T) {
	t.Parallel()

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
}

func TestUserJSONUnmarshaling(t *testing.T) {
	t.Parallel()

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
}

func TestCreateUserRequest(t *testing.T) {
	t.Parallel()

	// Test struct creation
	req := CreateUserRequest{
		Username: "newuser",
	}

	if req.Username != "newuser" {
		t.Errorf("Expected Username to be 'newuser', got %s", req.Username)
	}
}

func TestCreateUserRequestZeroValues(t *testing.T) {
	t.Parallel()

	var req CreateUserRequest

	if req.Username != "" {
		t.Errorf("Expected empty Username, got %s", req.Username)
	}
}

func TestCreateUserRequestJSONMarshaling(t *testing.T) {
	t.Parallel()

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
}

func TestCreateUserResponse(t *testing.T) {
	t.Parallel()

	// Test struct creation
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
}

func TestCreateUserResponseJSONMarshaling(t *testing.T) {
	t.Parallel()

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
}

func TestCreateUserResponseJSONUnmarshaling(t *testing.T) {
	t.Parallel()

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
}
