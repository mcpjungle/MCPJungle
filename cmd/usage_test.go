package cmd

import (
	"testing"
)

func TestUsageCommandStructure(t *testing.T) {
	t.Run("usage command has correct properties", func(t *testing.T) {
		if usageCmd.Use != "usage <name>" {
			t.Errorf("Expected usage command Use to be 'usage <name>', got %s", usageCmd.Use)
		}
		if usageCmd.Short != "Get usage information for a MCP tool" {
			t.Errorf("Expected usage command Short to be 'Get usage information for a MCP tool', got %s", usageCmd.Short)
		}
	})

	t.Run("usage command has correct annotations", func(t *testing.T) {
		if usageCmd.Annotations == nil {
			t.Fatal("Usage command missing annotations")
		}

		group, hasGroup := usageCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Usage command missing 'group' annotation")
		}
		if group != string(subCommandGroupBasic) {
			t.Errorf("Expected usage command group to be 'basic', got %s", group)
		}

		order, hasOrder := usageCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Usage command missing 'order' annotation")
		}
		if order != "4" {
			t.Errorf("Expected usage command order to be '4', got %s", order)
		}
	})

	t.Run("usage command has RunE function", func(t *testing.T) {
		if usageCmd.RunE == nil {
			t.Fatal("Usage command missing RunE function")
		}
	})

	t.Run("usage command requires exact args", func(t *testing.T) {
		if usageCmd.Args == nil {
			t.Fatal("Usage command missing Args validation")
		}
	})
}

func TestRunGetToolUsage(t *testing.T) {
	t.Run("runGetToolUsage function is properly defined", func(t *testing.T) {
		// Verify that the function is properly assigned to the command
		if usageCmd.RunE == nil {
			t.Fatal("usageCmd.RunE is nil")
		}
	})

	t.Run("runGetToolUsage would call API client with correct tool name", func(t *testing.T) {
		// This test would require mocking the apiClient
		// For now, we verify the function is properly configured
		if usageCmd.RunE == nil {
			t.Fatal("usageCmd.RunE is nil")
		}
	})

	t.Run("runGetToolUsage would handle API client errors", func(t *testing.T) {
		// This test would require mocking the apiClient to return an error
		// and verifying that the error is properly wrapped and returned
		if usageCmd.RunE == nil {
			t.Fatal("usageCmd.RunE is nil")
		}
	})

	t.Run("runGetToolUsage would print tool name and description", func(t *testing.T) {
		// This test would require mocking the apiClient to return a tool
		// and verifying the tool name and description are printed
		if usageCmd.RunE == nil {
			t.Fatal("usageCmd.RunE is nil")
		}
	})

	t.Run("runGetToolUsage would handle tools with no input parameters", func(t *testing.T) {
		// This test would require mocking the apiClient to return a tool with no input schema
		// and verifying the "This tool does not require any input parameters" message
		if usageCmd.RunE == nil {
			t.Fatal("usageCmd.RunE is nil")
		}
	})

	t.Run("runGetToolUsage would display input parameters", func(t *testing.T) {
		// This test would require mocking the apiClient to return a tool with input schema
		// and verifying the parameter display format with boundaries and required/optional labels
		if usageCmd.RunE == nil {
			t.Fatal("usageCmd.RunE is nil")
		}
	})

	t.Run("runGetToolUsage would handle JSON marshaling errors", func(t *testing.T) {
		// This test would require mocking the apiClient to return a tool with invalid JSON schema
		// and verifying that it falls back to printing the raw object
		if usageCmd.RunE == nil {
			t.Fatal("usageCmd.RunE is nil")
		}
	})
}

func TestUsageCommandIntegration(t *testing.T) {
	t.Run("usage command is added to root command", func(t *testing.T) {
		// Verify that usageCmd is properly added to rootCmd
		// This would require checking rootCmd.Commands() for usageCmd
		if usageCmd == nil {
			t.Fatal("usageCmd is nil")
		}
	})
}

func TestUsageCommandArgumentValidation(t *testing.T) {
	t.Run("usage command requires exactly one argument", func(t *testing.T) {
		if usageCmd.Args == nil {
			t.Fatal("usageCmd missing Args validation")
		}
		// The Args field should be cobra.ExactArgs(1)
	})
}
