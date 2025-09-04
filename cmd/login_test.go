package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestLoginCommandStructure(t *testing.T) {
	t.Run("login command has correct properties", func(t *testing.T) {
		if loginCmd.Use != "login [access_token]" {
			t.Errorf("Expected login command Use to be 'login [access_token]', got %s", loginCmd.Use)
		}
		if loginCmd.Short != "Log in to MCPJungle (Production mode)" {
			t.Errorf("Expected login command Short to be 'Log in to MCPJungle (Production mode)', got %s", loginCmd.Short)
		}
	})

	t.Run("login command has correct annotations", func(t *testing.T) {
		if loginCmd.Annotations == nil {
			t.Fatal("Login command missing annotations")
		}

		group, hasGroup := loginCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Login command missing 'group' annotation")
		}
		if group != string(subCommandGroupAdvanced) {
			t.Errorf("Expected login command group to be 'advanced', got %s", group)
		}

		order, hasOrder := loginCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Login command missing 'order' annotation")
		}
		if order != "6" {
			t.Errorf("Expected login command order to be '6', got %s", order)
		}
	})

	t.Run("login command requires exact args", func(t *testing.T) {
		if loginCmd.Args == nil {
			t.Fatal("Login command missing Args validation")
		}
	})
}

func TestUserRoleConstants(t *testing.T) {
	t.Run("UserRole constants match expected values", func(t *testing.T) {
		if string(types.UserRoleAdmin) != "admin" {
			t.Errorf("Expected UserRoleAdmin to be 'admin', got %s", string(types.UserRoleAdmin))
		}
		if string(types.UserRoleUser) != "user" {
			t.Errorf("Expected UserRoleUser to be 'user', got %s", string(types.UserRoleUser))
		}
	})
}
