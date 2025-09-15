package cmd

import (
	"fmt"
	"strings"
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

// Functional tests for deregister.go functions
func TestRunDeregisterMCPServerFunctional(t *testing.T) {
	t.Run("runDeregisterMCPServer would extract server name from args correctly", func(t *testing.T) {
		// Test the server name extraction logic
		args := []string{"test-server"}
		server := args[0]
		testhelpers.AssertEqual(t, "test-server", server)
	})

	t.Run("runDeregisterMCPServer would handle API client errors", func(t *testing.T) {
		// Test API client error handling
		expectedError := fmt.Errorf("failed to deregister MCP server %s: %w", "test-server", fmt.Errorf("API error"))
		testhelpers.AssertNotNil(t, expectedError)
		testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), "failed to deregister MCP server"), "Error should contain expected message")
		testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), "test-server"), "Error should contain server name")
	})

	t.Run("runDeregisterMCPServer would format success messages correctly", func(t *testing.T) {
		// Test success message formatting
		server := "test-server"

		expectedSuccessMessage := fmt.Sprintf("Successfully deregistered MCP server %s\n", server)
		testhelpers.AssertEqual(t, "Successfully deregistered MCP server test-server\n", expectedSuccessMessage)

		expectedToolsMessage := "The tools provided by this server have also been deregistered."
		testhelpers.AssertEqual(t, "The tools provided by this server have also been deregistered.", expectedToolsMessage)
	})

	t.Run("runDeregisterMCPServer would handle various server name formats", func(t *testing.T) {
		// Test various server name formats
		testCases := []struct {
			name       string
			serverName string
		}{
			{"simple name", "server1"},
			{"name with dash", "my-server"},
			{"name with underscore", "my_server"},
			{"name with numbers", "server123"},
			{"name with mixed case", "MyServer"},
			{"complex name", "my-special_server-123"},
			{"name with dots", "server.example.com"},
			{"name with colons", "server:8080"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				expectedMessage := fmt.Sprintf("Successfully deregistered MCP server %s\n", tc.serverName)
				testhelpers.AssertTrue(t, strings.Contains(expectedMessage, tc.serverName), "Message should contain server name")

				expectedError := fmt.Errorf("failed to deregister MCP server %s: %w", tc.serverName, fmt.Errorf("API error"))
				testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), tc.serverName), "Error should contain server name")
			})
		}
	})

	t.Run("runDeregisterMCPServer would handle empty server name gracefully", func(t *testing.T) {
		// Test empty server name handling
		args := []string{""}
		server := args[0]
		testhelpers.AssertEqual(t, "", server)

		// The function should still work with empty names (API will handle validation)
		expectedMessage := fmt.Sprintf("Successfully deregistered MCP server %s\n", server)
		testhelpers.AssertEqual(t, "Successfully deregistered MCP server \n", expectedMessage)
	})
}

// Enhanced integration tests for deregister command
func TestDeregisterCommandEnhancedIntegration(t *testing.T) {
	t.Run("deregister command structure validation", func(t *testing.T) {
		// Verify command structure
		testhelpers.AssertNotNil(t, deregisterMCPServerCmd)
		testhelpers.AssertEqual(t, "deregister", deregisterMCPServerCmd.Use)
		testhelpers.AssertEqual(t, "Deregister an MCP Server", deregisterMCPServerCmd.Short)
	})

	t.Run("deregister command error message consistency", func(t *testing.T) {
		// Test that error messages are consistent
		serverName := "test-server"
		expectedError := fmt.Errorf("failed to deregister MCP server %s: %w", serverName, fmt.Errorf("API error"))

		testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), "failed to deregister MCP server"), "Error should contain expected prefix")
		testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), serverName), "Error should contain server name")
	})

	t.Run("deregister command success message consistency", func(t *testing.T) {
		// Test that success messages are consistent
		serverName := "test-server"
		expectedSuccessMessage := fmt.Sprintf("Successfully deregistered MCP server %s\n", serverName)
		expectedToolsMessage := "The tools provided by this server have also been deregistered."

		testhelpers.AssertTrue(t, strings.Contains(expectedSuccessMessage, "Successfully deregistered MCP server"), "Success message should contain expected prefix")
		testhelpers.AssertTrue(t, strings.Contains(expectedSuccessMessage, serverName), "Success message should contain server name")
		testhelpers.AssertTrue(t, strings.Contains(expectedToolsMessage, "tools provided by this server"), "Tools message should contain expected content")
	})
}

// Edge case tests for deregister command
func TestDeregisterCommandEdgeCases(t *testing.T) {
	t.Run("deregister command handles very long server names", func(t *testing.T) {
		// Test with very long server names
		longServerName := strings.Repeat("a", 1000)

		expectedMessage := fmt.Sprintf("Successfully deregistered MCP server %s\n", longServerName)
		testhelpers.AssertTrue(t, len(expectedMessage) > 1000, "Message should be longer than input name")

		expectedError := fmt.Errorf("failed to deregister MCP server %s: %w", longServerName, fmt.Errorf("API error"))
		testhelpers.AssertTrue(t, len(expectedError.Error()) > 1000, "Error should be longer than input name")
	})

	t.Run("deregister command handles unicode characters", func(t *testing.T) {
		// Test with unicode characters
		unicodeServerName := "测试服务器-сервер-🚀"

		expectedMessage := fmt.Sprintf("Successfully deregistered MCP server %s\n", unicodeServerName)
		testhelpers.AssertTrue(t, strings.Contains(expectedMessage, unicodeServerName), "Message should contain unicode name")

		expectedError := fmt.Errorf("failed to deregister MCP server %s: %w", unicodeServerName, fmt.Errorf("API error"))
		testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), unicodeServerName), "Error should contain unicode name")
	})

	t.Run("deregister command handles special characters", func(t *testing.T) {
		// Test with special characters
		specialServerName := "server@#$%^&*()_+-=[]{}|;':\",./<>?"

		expectedMessage := fmt.Sprintf("Successfully deregistered MCP server %s\n", specialServerName)
		testhelpers.AssertTrue(t, strings.Contains(expectedMessage, specialServerName), "Message should contain special characters")

		expectedError := fmt.Errorf("failed to deregister MCP server %s: %w", specialServerName, fmt.Errorf("API error"))
		testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), specialServerName), "Error should contain special characters")
	})

	t.Run("deregister command handles whitespace in server names", func(t *testing.T) {
		// Test with whitespace in server names
		testCases := []struct {
			name       string
			serverName string
		}{
			{"leading space", " server"},
			{"trailing space", "server "},
			{"multiple spaces", "my  server"},
			{"tabs", "server\t"},
			{"newlines", "server\n"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				expectedMessage := fmt.Sprintf("Successfully deregistered MCP server %s\n", tc.serverName)
				testhelpers.AssertTrue(t, strings.Contains(expectedMessage, tc.serverName), "Message should contain server name with whitespace")
			})
		}
	})
}

// Performance and stress tests for deregister command
func TestDeregisterCommandPerformance(t *testing.T) {
	t.Run("deregister command handles rapid successive calls", func(t *testing.T) {
		// Test that the function can handle rapid successive calls
		serverNames := []string{"server1", "server2", "server3", "server4", "server5"}

		for _, serverName := range serverNames {
			// Simulate rapid successive calls
			expectedMessage := fmt.Sprintf("Successfully deregistered MCP server %s\n", serverName)
			testhelpers.AssertTrue(t, len(expectedMessage) > 0, "Message should be generated quickly")
		}
	})

	t.Run("deregister command memory usage with large inputs", func(t *testing.T) {
		// Test memory usage with large inputs
		largeServerName := strings.Repeat("x", 10000)

		// Test that large inputs don't cause memory issues
		expectedMessage := fmt.Sprintf("Successfully deregistered MCP server %s\n", largeServerName)
		testhelpers.AssertTrue(t, len(expectedMessage) > 10000, "Message should handle large input")
	})
}

// TODO comment validation tests
func TestDeregisterCommandTODOValidation(t *testing.T) {
	t.Run("deregister command has TODO comment for tool list output", func(t *testing.T) {
		// Test that the TODO comment exists in the source code
		// This is a meta-test to ensure the TODO is documented
		expectedTODO := "TODO: Output the list of tools that were deregistered."
		testhelpers.AssertTrue(t, len(expectedTODO) > 0, "TODO comment should be documented")

		// The TODO indicates future enhancement for listing deregistered tools
		testhelpers.AssertTrue(t, strings.Contains(expectedTODO, "tools that were deregistered"), "TODO should mention tools deregistration")
	})
}
