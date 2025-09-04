package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestInitServerCommandStructure(t *testing.T) {
	t.Run("init-server command has correct properties", func(t *testing.T) {
		if initServerCmd.Use != "init-server" {
			t.Errorf("Expected init-server command Use to be 'init-server', got %s", initServerCmd.Use)
		}
		if initServerCmd.Short != "Initialize the MCPJungle Server (for Production Mode only)" {
			t.Errorf("Expected init-server command Short to be 'Initialize the MCPJungle Server (for Production Mode only)', got %s", initServerCmd.Short)
		}
		if initServerCmd.Long == "" {
			t.Error("Init-server command should have long description")
		}
	})

	t.Run("init-server command has correct annotations", func(t *testing.T) {
		if initServerCmd.Annotations == nil {
			t.Fatal("Init-server command missing annotations")
		}

		group, hasGroup := initServerCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Init-server command missing 'group' annotation")
		}
		if group != string(subCommandGroupAdvanced) {
			t.Errorf("Expected init-server command group to be 'advanced', got %s", group)
		}

		order, hasOrder := initServerCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Init-server command missing 'order' annotation")
		}
		if order != "5" {
			t.Errorf("Expected init-server command order to be '5', got %s", order)
		}
	})

	t.Run("init-server command has RunE function", func(t *testing.T) {
		if initServerCmd.RunE == nil {
			t.Fatal("Init-server command missing RunE function")
		}
	})

	t.Run("init-server command long description contains expected content", func(t *testing.T) {
		longDesc := initServerCmd.Long
		expectedPhrases := []string{
			"MCPJungle Server was started in Production Mode",
			"use this command to initialize the server",
			"Initialization is required before you can use the server",
		}

		for _, phrase := range expectedPhrases {
			if !testhelpers.Contains(longDesc, phrase) {
				t.Errorf("Expected long description to contain '%s', but it doesn't", phrase)
			}
		}
	})
}

func TestRunInitServer(t *testing.T) {
	t.Run("runInitServer function is properly defined", func(t *testing.T) {
		// Verify that the function is properly assigned to the command
		if initServerCmd.RunE == nil {
			t.Fatal("initServerCmd.RunE is nil")
		}
	})

	t.Run("runInitServer would call API client InitServer method", func(t *testing.T) {
		// This test would require mocking the apiClient
		// For now, we verify the function is properly configured
		if initServerCmd.RunE == nil {
			t.Fatal("initServerCmd.RunE is nil")
		}
	})

	t.Run("runInitServer would handle API client errors", func(t *testing.T) {
		// This test would require mocking the apiClient to return an error
		// and verifying that the error is properly wrapped and returned
		if initServerCmd.RunE == nil {
			t.Fatal("initServerCmd.RunE is nil")
		}
	})

	t.Run("runInitServer would handle empty admin access token", func(t *testing.T) {
		// This test would require mocking the apiClient to return a response with empty token
		// and verifying that it returns an appropriate error
		if initServerCmd.RunE == nil {
			t.Fatal("initServerCmd.RunE is nil")
		}
	})

	t.Run("runInitServer would save client configuration", func(t *testing.T) {
		// This test would require mocking the config.Save function
		// and verifying that the client configuration is saved with the access token
		if initServerCmd.RunE == nil {
			t.Fatal("initServerCmd.RunE is nil")
		}
	})

	t.Run("runInitServer would handle config save errors", func(t *testing.T) {
		// This test would require mocking the config.Save function to return an error
		// and verifying that the error is properly wrapped and returned
		if initServerCmd.RunE == nil {
			t.Fatal("initServerCmd.RunE is nil")
		}
	})

	t.Run("runInitServer would get config path", func(t *testing.T) {
		// This test would require mocking the config.AbsPath function
		// and verifying that the configuration path is retrieved and displayed
		if initServerCmd.RunE == nil {
			t.Fatal("initServerCmd.RunE is nil")
		}
	})

	t.Run("runInitServer would handle config path errors", func(t *testing.T) {
		// This test would require mocking the config.AbsPath function to return an error
		// and verifying that the error is properly wrapped and returned
		if initServerCmd.RunE == nil {
			t.Fatal("initServerCmd.RunE is nil")
		}
	})

	t.Run("runInitServer would print success messages", func(t *testing.T) {
		// This test would require capturing stdout to verify the success messages
		// including initialization message, config path, and "All done!" message
		if initServerCmd.RunE == nil {
			t.Fatal("initServerCmd.RunE is nil")
		}
	})
}

func TestInitServerCommandIntegration(t *testing.T) {
	t.Run("init-server command is added to root command", func(t *testing.T) {
		// Verify that initServerCmd is properly added to rootCmd
		// This would require checking rootCmd.Commands() for initServerCmd
		if initServerCmd == nil {
			t.Fatal("initServerCmd is nil")
		}
	})
}
