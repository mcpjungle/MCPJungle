package user

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestNewUserService(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	svc := NewUserService(db)
	service.AssertNotNil(t, svc)
	if svc.db != db {
		t.Errorf("Expected db to be %v, got %v", db, svc.db)
	}
}

func TestCreateUser(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the User model
	err = db.AutoMigrate(&model.User{})
	service.AssertNoError(t, err)

	svc := NewUserService(db)

	username := "testuser"
	user, err := svc.CreateUser(username)
	service.AssertNoError(t, err)
	service.AssertNotNil(t, user)

	// Verify user properties
	service.AssertEqual(t, username, user.Username)
	service.AssertEqual(t, types.UserRoleUser, user.Role)
	if user.AccessToken == "" {
		t.Error("Expected access token to be generated")
	}
}

func TestCreateUserWithExistingUsername(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the User model
	err = db.AutoMigrate(&model.User{})
	service.AssertNoError(t, err)

	svc := NewUserService(db)

	username := "testuser"

	// Create first user
	user1, err := svc.CreateUser(username)
	service.AssertNoError(t, err)
	service.AssertNotNil(t, user1)

	// Try to create another user with same username
	user2, err := svc.CreateUser(username)
	service.AssertError(t, err)
	if user2 != nil {
		t.Error("Expected second user creation to fail")
	}
}

func TestCreateAdminUser(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the User model
	err = db.AutoMigrate(&model.User{})
	service.AssertNoError(t, err)

	svc := NewUserService(db)

	user, err := svc.CreateAdminUser()
	service.AssertNoError(t, err)
	service.AssertNotNil(t, user)

	// Verify admin user properties
	service.AssertEqual(t, "admin", user.Username)
	service.AssertEqual(t, types.UserRoleAdmin, user.Role)
	if user.AccessToken == "" {
		t.Error("Expected access token to be generated")
	}
}

func TestGetUserByAccessToken(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the User model
	err = db.AutoMigrate(&model.User{})
	service.AssertNoError(t, err)

	svc := NewUserService(db)

	// Create a test user first
	username := "testuser"
	user, err := svc.CreateUser(username)
	service.AssertNoError(t, err)

	// Test getting user by valid token
	retrievedUser, err := svc.GetUserByAccessToken(user.AccessToken)
	service.AssertNoError(t, err)
	service.AssertNotNil(t, retrievedUser)
	service.AssertEqual(t, username, retrievedUser.Username)
	service.AssertEqual(t, user.AccessToken, retrievedUser.AccessToken)

	// Test getting user by invalid token
	_, err = svc.GetUserByAccessToken("invalid-token")
	service.AssertError(t, err)
}

func TestListUsers(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the User model
	err = db.AutoMigrate(&model.User{})
	service.AssertNoError(t, err)

	svc := NewUserService(db)

	// Initially should be empty
	users, err := svc.ListUsers()
	service.AssertNoError(t, err)
	if len(users) != 0 {
		t.Errorf("Expected 0 users initially, got %d", len(users))
	}

	// Create some users
	_, err = svc.CreateUser("user1")
	service.AssertNoError(t, err)

	_, err = svc.CreateUser("user2")
	service.AssertNoError(t, err)

	// Now should have 2 users
	users, err = svc.ListUsers()
	service.AssertNoError(t, err)
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	// Verify all users are present
	usernames := make(map[string]bool)
	for _, user := range users {
		usernames[user.Username] = true
	}

	expectedUsernames := []string{"user1", "user2"}
	for _, expected := range expectedUsernames {
		if !usernames[expected] {
			t.Errorf("Expected user %s to be in list", expected)
		}
	}
}

func TestDeleteUser(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the User model
	err = db.AutoMigrate(&model.User{})
	service.AssertNoError(t, err)

	svc := NewUserService(db)

	// Create a test user
	username := "testuser"
	user, err := svc.CreateUser(username)
	service.AssertNoError(t, err)

	// Verify user exists
	_, err = svc.GetUserByAccessToken(user.AccessToken)
	service.AssertNoError(t, err)

	// Delete the user
	err = svc.DeleteUser(username)
	service.AssertNoError(t, err)

	// Verify user was deleted
	_, err = svc.GetUserByAccessToken(user.AccessToken)
	service.AssertError(t, err)
}

func TestDeleteUserNotFound(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the User model
	err = db.AutoMigrate(&model.User{})
	service.AssertNoError(t, err)

	svc := NewUserService(db)

	// Try to delete non-existent user
	err = svc.DeleteUser("nonexistent")
	service.AssertError(t, err)
}

func TestDeleteAdminUser(t *testing.T) {
	db, err := service.CreateTestDB()
	service.AssertNoError(t, err)

	// Auto-migrate the User model
	err = db.AutoMigrate(&model.User{})
	service.AssertNoError(t, err)

	svc := NewUserService(db)

	// Create admin user
	admin, err := svc.CreateAdminUser()
	service.AssertNoError(t, err)

	// Try to delete admin user (should fail)
	err = svc.DeleteUser("admin")
	service.AssertError(t, err)

	// Verify admin user still exists
	retrievedUser, err := svc.GetUserByAccessToken(admin.AccessToken)
	service.AssertNoError(t, err)
	service.AssertEqual(t, "admin", retrievedUser.Username)
}
