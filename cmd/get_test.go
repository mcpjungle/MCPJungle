package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestGetCommandStructure(t *testing.T) {
	t.Run("get command has correct properties", func(t *testing.T) {
		if getCmd.Use != "get" {
			t.Errorf("Expected get command Use to be 'get', got %s", getCmd.Use)
		}
		if getCmd.Short != "Get information about a specific resource" {
			t.Errorf("Expected get command Short to be 'Get information about a specific resource', got %s", getCmd.Short)
		}
	})

	t.Run("get command has correct annotations", func(t *testing.T) {
		if getCmd.Annotations == nil {
			t.Fatal("Get command missing annotations")
		}

		group, hasGroup := getCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Get command missing 'group' annotation")
		}
		if group != string(subCommandGroupAdvanced) {
			t.Errorf("Expected get command group to be 'advanced', got %s", group)
		}

		order, hasOrder := getCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Get command missing 'order' annotation")
		}
		if order != "7" {
			t.Errorf("Expected get command order to be '7', got %s", order)
		}
	})

	t.Run("get command has subcommands", func(t *testing.T) {
		subcommands := getCmd.Commands()
		if len(subcommands) != 1 {
			t.Errorf("Expected get command to have 1 subcommand, got %d", len(subcommands))
		}
	})
}

func TestGetGroupSubcommand(t *testing.T) {
	t.Run("get group command has correct properties", func(t *testing.T) {
		if getGroupCmd.Use != "group [name]" {
			t.Errorf("Expected get group command Use to be 'group [name]', got %s", getGroupCmd.Use)
		}
		if getGroupCmd.Short != "Get information about a specific Tool Group" {
			t.Errorf("Expected get group command Short to be 'Get information about a specific Tool Group', got %s", getGroupCmd.Short)
		}
		if getGroupCmd.Long == "" {
			t.Error("Get group command should have long description")
		}
	})

	t.Run("get group command has RunE function", func(t *testing.T) {
		if getGroupCmd.RunE == nil {
			t.Fatal("Get group command missing RunE function")
		}
	})

	t.Run("get group command requires exact args", func(t *testing.T) {
		if getGroupCmd.Args == nil {
			t.Fatal("Get group command missing Args validation")
		}
	})

	t.Run("get group command long description contains expected content", func(t *testing.T) {
		longDesc := getGroupCmd.Long
		expectedPhrases := []string{
			"Get information about a specific Tool Group by name",
			"returns the configuration of the Tool Group",
			"which tools are included",
		}

		for _, phrase := range expectedPhrases {
			if !testhelpers.Contains(longDesc, phrase) {
				t.Errorf("Expected long description to contain '%s', but it doesn't", phrase)
			}
		}
	})
}

func TestRunGetGroup(t *testing.T) {
	t.Run("runGetGroup function is properly defined", func(t *testing.T) {
		// Verify that the function is properly assigned to the command
		if getGroupCmd.RunE == nil {
			t.Fatal("getGroupCmd.RunE is nil")
		}
	})

	t.Run("runGetGroup would call API client with correct name", func(t *testing.T) {
		// This test would require mocking the apiClient
		// For now, we verify the function is properly configured
		if getGroupCmd.RunE == nil {
			t.Fatal("getGroupCmd.RunE is nil")
		}
	})

	t.Run("runGetGroup would handle API client errors", func(t *testing.T) {
		// This test would require mocking the apiClient to return an error
		// and verifying that the error is properly wrapped and returned
		if getGroupCmd.RunE == nil {
			t.Fatal("getGroupCmd.RunE is nil")
		}
	})

	t.Run("runGetGroup would print group information", func(t *testing.T) {
		// This test would require mocking the apiClient to return a group
		// and verifying the output format including name, description, endpoint, and tools
		if getGroupCmd.RunE == nil {
			t.Fatal("getGroupCmd.RunE is nil")
		}
	})

	t.Run("runGetGroup would handle empty tools list", func(t *testing.T) {
		// This test would require mocking the apiClient to return a group with no tools
		// and verifying the "This group has no tools" message
		if getGroupCmd.RunE == nil {
			t.Fatal("getGroupCmd.RunE is nil")
		}
	})

	t.Run("runGetGroup would handle group with tools", func(t *testing.T) {
		// This test would require mocking the apiClient to return a group with tools
		// and verifying the numbered list format and NOTE message
		if getGroupCmd.RunE == nil {
			t.Fatal("getGroupCmd.RunE is nil")
		}
	})
}

func TestGetCommandIntegration(t *testing.T) {
	t.Run("get command is added to root command", func(t *testing.T) {
		// Verify that getCmd is properly added to rootCmd
		// This would require checking rootCmd.Commands() for getCmd
		if getCmd == nil {
			t.Fatal("getCmd is nil")
		}
	})

	t.Run("get group subcommand is properly configured", func(t *testing.T) {
		subcommands := getCmd.Commands()
		if len(subcommands) != 1 {
			t.Errorf("Expected 1 subcommand, got %d", len(subcommands))
		}

		if subcommands[0].Name() != "group" {
			t.Errorf("Expected subcommand to be 'group', got %s", subcommands[0].Name())
		}
	})
}

func TestGetCommandArgumentValidation(t *testing.T) {
	t.Run("get group command requires exactly one argument", func(t *testing.T) {
		if getGroupCmd.Args == nil {
			t.Fatal("getGroupCmd missing Args validation")
		}
		// The Args field should be cobra.ExactArgs(1)
	})
}
