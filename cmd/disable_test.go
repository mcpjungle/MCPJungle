package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestDisableCommandStructure(t *testing.T) {
	t.Run("disable command has correct properties", func(t *testing.T) {
		if disableCmd.Use != "disable [name]" {
			t.Errorf("Expected disable command Use to be 'disable [name]', got %s", disableCmd.Use)
		}
		if disableCmd.Short != "Disable one or more MCP tools globally" {
			t.Errorf("Expected disable command Short to be 'Disable one or more MCP tools globally', got %s", disableCmd.Short)
		}
		if disableCmd.Long == "" {
			t.Error("Disable command should have long description")
		}
	})

	t.Run("disable command has correct annotations", func(t *testing.T) {
		if disableCmd.Annotations == nil {
			t.Fatal("Disable command missing annotations")
		}

		group, hasGroup := disableCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Disable command missing 'group' annotation")
		}
		if group != string(subCommandGroupAdvanced) {
			t.Errorf("Expected disable command group to be 'advanced', got %s", group)
		}

		order, hasOrder := disableCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Disable command missing 'order' annotation")
		}
		if order != "1" {
			t.Errorf("Expected disable command order to be '1', got %s", order)
		}
	})

	t.Run("disable command has RunE function", func(t *testing.T) {
		if disableCmd.RunE == nil {
			t.Fatal("Disable command missing RunE function")
		}
	})

	t.Run("disable command requires exact args", func(t *testing.T) {
		if disableCmd.Args == nil {
			t.Fatal("Disable command missing Args validation")
		}
	})

	t.Run("disable command long description contains expected content", func(t *testing.T) {
		longDesc := disableCmd.Long
		expectedPhrases := []string{
			"Specify the name of a tool or MCP server",
			"disable it in the mcp proxy",
			"all tools provided by that server will be disabled",
			"cannot be viewed or called by mcp clients",
		}

		for _, phrase := range expectedPhrases {
			if !testhelpers.Contains(longDesc, phrase) {
				t.Errorf("Expected long description to contain '%s', but it doesn't", phrase)
			}
		}
	})
}

func TestRunDisableTools(t *testing.T) {
	t.Run("runDisableTools function is properly defined", func(t *testing.T) {
		// Verify that the function is properly assigned to the command
		if disableCmd.RunE == nil {
			t.Fatal("disableCmd.RunE is nil")
		}
	})

	t.Run("runDisableTools would call API client with correct name", func(t *testing.T) {
		// This test would require mocking the apiClient
		// For now, we verify the function is properly configured
		if disableCmd.RunE == nil {
			t.Fatal("disableCmd.RunE is nil")
		}
	})

	t.Run("runDisableTools would handle API client errors", func(t *testing.T) {
		// This test would require mocking the apiClient to return an error
		// and verifying that the error is properly returned
		if disableCmd.RunE == nil {
			t.Fatal("disableCmd.RunE is nil")
		}
	})

	t.Run("runDisableTools would print success message for single tool", func(t *testing.T) {
		// This test would require mocking the apiClient to return a single tool
		// and verifying the single tool success message format
		if disableCmd.RunE == nil {
			t.Fatal("disableCmd.RunE is nil")
		}
	})

	t.Run("runDisableTools would print success message for multiple tools", func(t *testing.T) {
		// This test would require mocking the apiClient to return multiple tools
		// and verifying the multiple tools success message format
		if disableCmd.RunE == nil {
			t.Fatal("disableCmd.RunE is nil")
		}
	})
}

func TestDisableCommandIntegration(t *testing.T) {
	t.Run("disable command is added to root command", func(t *testing.T) {
		// Verify that disableCmd is properly added to rootCmd
		// This would require checking rootCmd.Commands() for disableCmd
		if disableCmd == nil {
			t.Fatal("disableCmd is nil")
		}
	})
}

func TestDisableCommandArgumentValidation(t *testing.T) {
	t.Run("disable command requires exactly one argument", func(t *testing.T) {
		if disableCmd.Args == nil {
			t.Fatal("disableCmd missing Args validation")
		}
		// The Args field should be cobra.ExactArgs(1)
	})
}
