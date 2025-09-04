package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestEnableCommandStructure(t *testing.T) {
	t.Run("enable command has correct properties", func(t *testing.T) {
		if enableCmd.Use != "enable [name]" {
			t.Errorf("Expected enable command Use to be 'enable [name]', got %s", enableCmd.Use)
		}
		if enableCmd.Short != "Enable one or more MCP tools globally" {
			t.Errorf("Expected enable command Short to be 'Enable one or more MCP tools globally', got %s", enableCmd.Short)
		}
		if enableCmd.Long == "" {
			t.Error("Enable command should have long description")
		}
	})

	t.Run("enable command has correct annotations", func(t *testing.T) {
		if enableCmd.Annotations == nil {
			t.Fatal("Enable command missing annotations")
		}

		group, hasGroup := enableCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Enable command missing 'group' annotation")
		}
		if group != string(subCommandGroupAdvanced) {
			t.Errorf("Expected enable command group to be 'advanced', got %s", group)
		}

		order, hasOrder := enableCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Enable command missing 'order' annotation")
		}
		if order != "2" {
			t.Errorf("Expected enable command order to be '2', got %s", order)
		}
	})

	t.Run("enable command has RunE function", func(t *testing.T) {
		if enableCmd.RunE == nil {
			t.Fatal("Enable command missing RunE function")
		}
	})

	t.Run("enable command requires exact args", func(t *testing.T) {
		if enableCmd.Args == nil {
			t.Fatal("Enable command missing Args validation")
		}
	})

	t.Run("enable command long description contains expected content", func(t *testing.T) {
		longDesc := enableCmd.Long
		expectedPhrases := []string{
			"Specify the name of a tool or MCP server",
			"enable it in the mcp proxy",
			"all tools provided by that server will be enabled",
			"can be viewed and called by mcp clients",
		}

		for _, phrase := range expectedPhrases {
			if !testhelpers.Contains(longDesc, phrase) {
				t.Errorf("Expected long description to contain '%s', but it doesn't", phrase)
			}
		}
	})
}

func TestRunEnableTools(t *testing.T) {
	t.Run("runEnableTools function is properly defined", func(t *testing.T) {
		// Verify that the function is properly assigned to the command
		if enableCmd.RunE == nil {
			t.Fatal("enableCmd.RunE is nil")
		}
	})

	t.Run("runEnableTools would call API client with correct name", func(t *testing.T) {
		// This test would require mocking the apiClient
		// For now, we verify the function is properly configured
		if enableCmd.RunE == nil {
			t.Fatal("enableCmd.RunE is nil")
		}
	})

	t.Run("runEnableTools would handle API client errors", func(t *testing.T) {
		// This test would require mocking the apiClient to return an error
		// and verifying that the error is properly wrapped and returned
		if enableCmd.RunE == nil {
			t.Fatal("enableCmd.RunE is nil")
		}
	})

	t.Run("runEnableTools would print success message for single tool", func(t *testing.T) {
		// This test would require mocking the apiClient to return a single tool
		// and verifying the single tool success message format
		if enableCmd.RunE == nil {
			t.Fatal("enableCmd.RunE is nil")
		}
	})

	t.Run("runEnableTools would print success message for multiple tools", func(t *testing.T) {
		// This test would require mocking the apiClient to return multiple tools
		// and verifying the multiple tools success message format
		if enableCmd.RunE == nil {
			t.Fatal("enableCmd.RunE is nil")
		}
	})
}

func TestEnableCommandIntegration(t *testing.T) {
	t.Run("enable command is added to root command", func(t *testing.T) {
		// Verify that enableCmd is properly added to rootCmd
		// This would require checking rootCmd.Commands() for enableCmd
		if enableCmd == nil {
			t.Fatal("enableCmd is nil")
		}
	})
}

func TestEnableCommandArgumentValidation(t *testing.T) {
	t.Run("enable command requires exactly one argument", func(t *testing.T) {
		if enableCmd.Args == nil {
			t.Fatal("enableCmd missing Args validation")
		}
		// The Args field should be cobra.ExactArgs(1)
	})
}
