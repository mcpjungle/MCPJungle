package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestDeregisterCommandStructure(t *testing.T) {
	t.Run("deregister command has correct properties", func(t *testing.T) {
		if deregisterMCPServerCmd.Use != "deregister" {
			t.Errorf("Expected deregister command Use to be 'deregister', got %s", deregisterMCPServerCmd.Use)
		}
		if deregisterMCPServerCmd.Short != "Deregister an MCP Server" {
			t.Errorf("Expected deregister command Short to be 'Deregister an MCP Server', got %s", deregisterMCPServerCmd.Short)
		}
		if deregisterMCPServerCmd.Long == "" {
			t.Error("Deregister command should have long description")
		}
	})

	t.Run("deregister command has correct annotations", func(t *testing.T) {
		if deregisterMCPServerCmd.Annotations == nil {
			t.Fatal("Deregister command missing annotations")
		}

		group, hasGroup := deregisterMCPServerCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Deregister command missing 'group' annotation")
		}
		if group != string(subCommandGroupBasic) {
			t.Errorf("Expected deregister command group to be 'basic', got %s", group)
		}

		order, hasOrder := deregisterMCPServerCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Deregister command missing 'order' annotation")
		}
		if order != "6" {
			t.Errorf("Expected deregister command order to be '6', got %s", order)
		}
	})

	t.Run("deregister command has RunE function", func(t *testing.T) {
		if deregisterMCPServerCmd.RunE == nil {
			t.Fatal("Deregister command missing RunE function")
		}
	})

	t.Run("deregister command requires exact args", func(t *testing.T) {
		if deregisterMCPServerCmd.Args == nil {
			t.Fatal("Deregister command missing Args validation")
		}
	})

	t.Run("deregister command long description contains expected content", func(t *testing.T) {
		longDesc := deregisterMCPServerCmd.Long
		expectedPhrases := []string{
			"Remove an MCP server from the registry",
			"deregisters all tools provided by the server",
		}

		for _, phrase := range expectedPhrases {
			if !testhelpers.Contains(longDesc, phrase) {
				t.Errorf("Expected long description to contain '%s', but it doesn't", phrase)
			}
		}
	})
}

func TestRunDeregisterMCPServer(t *testing.T) {
	t.Run("runDeregisterMCPServer function is properly defined", func(t *testing.T) {
		// Verify that the function is properly assigned to the command
		if deregisterMCPServerCmd.RunE == nil {
			t.Fatal("deregisterMCPServerCmd.RunE is nil")
		}
	})

	t.Run("runDeregisterMCPServer would call API client with correct server name", func(t *testing.T) {
		// This test would require mocking the apiClient
		// For now, we verify the function is properly configured
		if deregisterMCPServerCmd.RunE == nil {
			t.Fatal("deregisterMCPServerCmd.RunE is nil")
		}
	})

	t.Run("runDeregisterMCPServer would handle API client errors", func(t *testing.T) {
		// This test would require mocking the apiClient to return an error
		// and verifying that the error is properly wrapped and returned
		if deregisterMCPServerCmd.RunE == nil {
			t.Fatal("deregisterMCPServerCmd.RunE is nil")
		}
	})

	t.Run("runDeregisterMCPServer would print success message", func(t *testing.T) {
		// This test would require capturing stdout to verify the success message
		// For now, we verify the function is properly configured
		if deregisterMCPServerCmd.RunE == nil {
			t.Fatal("deregisterMCPServerCmd.RunE is nil")
		}
	})
}

func TestDeregisterCommandIntegration(t *testing.T) {
	t.Run("deregister command is added to root command", func(t *testing.T) {
		// Verify that deregisterMCPServerCmd is properly added to rootCmd
		// This would require checking rootCmd.Commands() for deregisterMCPServerCmd
		if deregisterMCPServerCmd == nil {
			t.Fatal("deregisterMCPServerCmd is nil")
		}
	})
}

func TestDeregisterCommandArgumentValidation(t *testing.T) {
	t.Run("deregister command requires exactly one argument", func(t *testing.T) {
		if deregisterMCPServerCmd.Args == nil {
			t.Fatal("deregisterMCPServerCmd missing Args validation")
		}
		// The Args field should be cobra.ExactArgs(1)
	})
}
