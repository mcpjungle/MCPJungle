package user

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestNewUserService(t *testing.T) {
	setup, _ := testhelpers.SetupUserTest(t)
	defer setup.Cleanup()
	svc := NewUserService(setup.DB)
	testhelpers.AssertNotNil(t, svc)
	testhelpers.AssertEqual(t, setup.DB, svc.db)
}

func TestCreateUser(t *testing.T) {
	setup, _ := testhelpers.SetupUserTest(t)
	defer setup.Cleanup()
	svc := NewUserService(setup.DB)
	username := "testuser2"
	user, err := svc.CreateUser(username)
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, user)
	// Verify user properties
	testhelpers.AssertEqual(t, username, user.Username)
	testhelpers.AssertEqual(t, types.UserRoleUser, user.Role)
	if user.AccessToken == "" {
		t.Error("Expected access token to be generated")
	}
}

func TestCreateUserWithExistingUsername(t *testing.T) {
	setup, _ := testhelpers.SetupUserTest(t)
	defer setup.Cleanup()
	svc := NewUserService(setup.DB)
	username := "testuser2"
	// Create first user
	user1, _ := svc.CreateUser(username)
	testhelpers.AssertNotNil(t, user1)
	// Try to create another user with same username
	user2, err := svc.CreateUser(username)
	testhelpers.AssertError(t, err)
	if user2 != nil {
		t.Error("Expected second user creation to fail")
	}
}

func TestCreateAdminUser(t *testing.T) {
	setup, _ := testhelpers.SetupUserTest(t)
	defer setup.Cleanup()
	svc := NewUserService(setup.DB)
	user, err := svc.CreateAdminUser()
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, user)
	// Verify admin user properties
	testhelpers.AssertEqual(t, "admin", user.Username)
	testhelpers.AssertEqual(t, types.UserRoleAdmin, user.Role)
	if user.AccessToken == "" {
		t.Error("Expected access token to be generated")
	}
}

func TestGetUserByAccessToken(t *testing.T) {
	setup, _ := testhelpers.SetupUserTest(t)
	defer setup.Cleanup()
	svc := NewUserService(setup.DB)
	// Create a test user first
	username := "testuser2"
	user, _ := svc.CreateUser(username)
	// Test getting user by valid token
	retrievedUser, _ := svc.GetUserByAccessToken(user.AccessToken)
	testhelpers.AssertNotNil(t, retrievedUser)
	testhelpers.AssertEqual(t, username, retrievedUser.Username)
	testhelpers.AssertEqual(t, user.AccessToken, retrievedUser.AccessToken)
	// Test getting user by invalid token
	_, err := svc.GetUserByAccessToken("invalid-token")
	testhelpers.AssertError(t, err)
}

func TestListUsers(t *testing.T) {
	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()
	svc := NewUserService(setup.DB)
	// Initially should be empty
	users, err := svc.ListUsers()
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertEqual(t, 0, len(users))
	// Create some users
	_, _ = svc.CreateUser("user1")
	_, _ = svc.CreateUser("user2")
	// Now should have 2 users
	users, _ = svc.ListUsers()
	testhelpers.AssertEqual(t, 2, len(users))
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
	setup, _ := testhelpers.SetupUserTest(t)
	defer setup.Cleanup()
	svc := NewUserService(setup.DB)
	// Create a test user
	username := "testuser2"
	user, _ := svc.CreateUser(username)
	// Verify user exists
	_, err := svc.GetUserByAccessToken(user.AccessToken)
	testhelpers.AssertNoError(t, err)
	// Delete the user
	err = svc.DeleteUser(username)
	testhelpers.AssertNoError(t, err)
	// Verify user was deleted
	_, err = svc.GetUserByAccessToken(user.AccessToken)
	testhelpers.AssertError(t, err)
}

func TestDeleteUserNotFound(t *testing.T) {
	setup, _ := testhelpers.SetupUserTest(t)
	defer setup.Cleanup()
	svc := NewUserService(setup.DB)
	// Try to delete non-existent user
	err := svc.DeleteUser("nonexistent")
	testhelpers.AssertError(t, err)
}

func TestDeleteAdminUser(t *testing.T) {
	setup, _ := testhelpers.SetupUserTest(t)
	defer setup.Cleanup()
	svc := NewUserService(setup.DB)
	// Create admin user
	admin, _ := svc.CreateAdminUser()
	// Try to delete admin user (should fail)
	err := svc.DeleteUser("admin")
	testhelpers.AssertError(t, err)
	// Verify admin user still exists
	retrievedUser, _ := svc.GetUserByAccessToken(admin.AccessToken)
	testhelpers.AssertEqual(t, "admin", retrievedUser.Username)
}
