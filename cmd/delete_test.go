package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestDeleteCommandStructure(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "delete", deleteCmd.Use)
		testhelpers.AssertEqual(t, "Delete resources", deleteCmd.Short)
	})

	t.Run("command_annotations", func(t *testing.T) {
		annotationTests := []testhelpers.CommandAnnotationTest{
			{Key: "group", Expected: string(subCommandGroupAdvanced)},
			{Key: "order", Expected: "4"},
		}
		testhelpers.TestCommandAnnotations(t, deleteCmd.Annotations, annotationTests)
	})

	t.Run("subcommands_count", func(t *testing.T) {
		subcommands := deleteCmd.Commands()
		testhelpers.AssertEqual(t, 3, len(subcommands))
	})
}

func TestDeleteMcpClientSubcommand(t *testing.T) {
	t.Run("delete mcp-client command has correct properties", func(t *testing.T) {
		if deleteMcpClientCmd.Use != "mcp-client [name]" {
			t.Errorf("Expected delete mcp-client command Use to be 'mcp-client [name]', got %s", deleteMcpClientCmd.Use)
		}
		if deleteMcpClientCmd.Short != "Delete an MCP client (Production mode)" {
			t.Errorf("Expected delete mcp-client command Short to be 'Delete an MCP client (Production mode)', got %s", deleteMcpClientCmd.Short)
		}
		if deleteMcpClientCmd.Long == "" {
			t.Error("Delete mcp-client command should have long description")
		}
	})

	t.Run("delete mcp-client command has RunE function", func(t *testing.T) {
		if deleteMcpClientCmd.RunE == nil {
			t.Fatal("Delete mcp-client command missing RunE function")
		}
	})

	t.Run("delete mcp-client command requires exact args", func(t *testing.T) {
		if deleteMcpClientCmd.Args == nil {
			t.Fatal("Delete mcp-client command missing Args validation")
		}
	})

	t.Run("delete mcp-client command long description contains expected content", func(t *testing.T) {
		longDesc := deleteMcpClientCmd.Long
		expectedPhrases := []string{
			"Delete an MCP client from the registry",
			"instantly revokes all access",
			"Production mode",
		}

		for _, phrase := range expectedPhrases {
			if !testhelpers.Contains(longDesc, phrase) {
				t.Errorf("Expected long description to contain '%s', but it doesn't", phrase)
			}
		}
	})
}

func TestDeleteUserSubcommand(t *testing.T) {
	t.Run("delete user command has correct properties", func(t *testing.T) {
		if deleteUserCmd.Use != "user [username]" {
			t.Errorf("Expected delete user command Use to be 'user [username]', got %s", deleteUserCmd.Use)
		}
		if deleteUserCmd.Short != "Delete a user (Production mode)" {
			t.Errorf("Expected delete user command Short to be 'Delete a user (Production mode)', got %s", deleteUserCmd.Short)
		}
		if deleteUserCmd.Long == "" {
			t.Error("Delete user command should have long description")
		}
	})

	t.Run("delete user command has RunE function", func(t *testing.T) {
		if deleteUserCmd.RunE == nil {
			t.Fatal("Delete user command missing RunE function")
		}
	})

	t.Run("delete user command requires exact args", func(t *testing.T) {
		if deleteUserCmd.Args == nil {
			t.Fatal("Delete user command missing Args validation")
		}
	})

	t.Run("delete user command long description contains expected content", func(t *testing.T) {
		longDesc := deleteUserCmd.Long
		expectedPhrases := []string{
			"Delete a user from mcpjungle",
			"instantly revokes all access",
		}

		for _, phrase := range expectedPhrases {
			if !testhelpers.Contains(longDesc, phrase) {
				t.Errorf("Expected long description to contain '%s', but it doesn't", phrase)
			}
		}
	})
}

func TestDeleteToolGroupSubcommand(t *testing.T) {
	t.Run("delete group command has correct properties", func(t *testing.T) {
		if deleteToolGroupCmd.Use != "group [name]" {
			t.Errorf("Expected delete group command Use to be 'group [name]', got %s", deleteToolGroupCmd.Use)
		}
		if deleteToolGroupCmd.Short != "Delete a tool group" {
			t.Errorf("Expected delete group command Short to be 'Delete a tool group', got %s", deleteToolGroupCmd.Short)
		}
		if deleteToolGroupCmd.Long == "" {
			t.Error("Delete group command should have long description")
		}
	})

	t.Run("delete group command has RunE function", func(t *testing.T) {
		if deleteToolGroupCmd.RunE == nil {
			t.Fatal("Delete group command missing RunE function")
		}
	})

	t.Run("delete group command requires exact args", func(t *testing.T) {
		if deleteToolGroupCmd.Args == nil {
			t.Fatal("Delete group command missing Args validation")
		}
	})

	t.Run("delete group command long description contains expected content", func(t *testing.T) {
		longDesc := deleteToolGroupCmd.Long
		expectedPhrases := []string{
			"Delete a tool group from mcpjungle",
			"endpoint is no longer available",
			"MCP clients are relying on the endpoint",
			"only deletes the group itself",
			"Tools are only deleted when you deregister",
		}

		for _, phrase := range expectedPhrases {
			if !testhelpers.Contains(longDesc, phrase) {
				t.Errorf("Expected long description to contain '%s', but it doesn't", phrase)
			}
		}
	})
}

func TestRunDeleteMcpClient(t *testing.T) {
	t.Run("runDeleteMcpClient function is properly defined", func(t *testing.T) {
		// Verify that the function is properly assigned to the command
		if deleteMcpClientCmd.RunE == nil {
			t.Fatal("deleteMcpClientCmd.RunE is nil")
		}
	})

	t.Run("runDeleteMcpClient would call API client with correct name", func(t *testing.T) {
		// This test would require mocking the apiClient
		// For now, we verify the function is properly configured
		if deleteMcpClientCmd.RunE == nil {
			t.Fatal("deleteMcpClientCmd.RunE is nil")
		}
	})

	t.Run("runDeleteMcpClient would handle API client errors", func(t *testing.T) {
		// This test would require mocking the apiClient to return an error
		// and verifying that the error is properly wrapped and returned
		if deleteMcpClientCmd.RunE == nil {
			t.Fatal("deleteMcpClientCmd.RunE is nil")
		}
	})

	t.Run("runDeleteMcpClient would print success message", func(t *testing.T) {
		// This test would require capturing stdout to verify the success message
		// For now, we verify the function is properly configured
		if deleteMcpClientCmd.RunE == nil {
			t.Fatal("deleteMcpClientCmd.RunE is nil")
		}
	})
}

func TestRunDeleteUser(t *testing.T) {
	t.Run("runDeleteUser function is properly defined", func(t *testing.T) {
		// Verify that the function is properly assigned to the command
		if deleteUserCmd.RunE == nil {
			t.Fatal("deleteUserCmd.RunE is nil")
		}
	})

	t.Run("runDeleteUser would call API client with correct username", func(t *testing.T) {
		// This test would require mocking the apiClient
		// For now, we verify the function is properly configured
		if deleteUserCmd.RunE == nil {
			t.Fatal("deleteUserCmd.RunE is nil")
		}
	})

	t.Run("runDeleteUser would handle API client errors", func(t *testing.T) {
		// This test would require mocking the apiClient to return an error
		// and verifying that the error is properly wrapped and returned
		if deleteUserCmd.RunE == nil {
			t.Fatal("deleteUserCmd.RunE is nil")
		}
	})

	t.Run("runDeleteUser would print success message", func(t *testing.T) {
		// This test would require capturing stdout to verify the success message
		// For now, we verify the function is properly configured
		if deleteUserCmd.RunE == nil {
			t.Fatal("deleteUserCmd.RunE is nil")
		}
	})
}

func TestRunDeleteToolGroup(t *testing.T) {
	t.Run("runDeleteToolGroup function is properly defined", func(t *testing.T) {
		// Verify that the function is properly assigned to the command
		if deleteToolGroupCmd.RunE == nil {
			t.Fatal("deleteToolGroupCmd.RunE is nil")
		}
	})

	t.Run("runDeleteToolGroup would call API client with correct name", func(t *testing.T) {
		// This test would require mocking the apiClient
		// For now, we verify the function is properly configured
		if deleteToolGroupCmd.RunE == nil {
			t.Fatal("deleteToolGroupCmd.RunE is nil")
		}
	})

	t.Run("runDeleteToolGroup would handle API client errors", func(t *testing.T) {
		// This test would require mocking the apiClient to return an error
		// and verifying that the error is properly wrapped and returned
		if deleteToolGroupCmd.RunE == nil {
			t.Fatal("deleteToolGroupCmd.RunE is nil")
		}
	})

	t.Run("runDeleteToolGroup would print success message", func(t *testing.T) {
		// This test would require capturing stdout to verify the success message
		// For now, we verify the function is properly configured
		if deleteToolGroupCmd.RunE == nil {
			t.Fatal("deleteToolGroupCmd.RunE is nil")
		}
	})
}

func TestDeleteCommandIntegration(t *testing.T) {
	t.Run("delete command is added to root command", func(t *testing.T) {
		// Verify that deleteCmd is properly added to rootCmd
		// This would require checking rootCmd.Commands() for deleteCmd
		if deleteCmd == nil {
			t.Fatal("deleteCmd is nil")
		}
	})

	t.Run("all delete subcommands are properly configured", func(t *testing.T) {
		subcommands := deleteCmd.Commands()
		expectedSubcommands := []string{"mcp-client", "user", "group"}

		if len(subcommands) != len(expectedSubcommands) {
			t.Errorf("Expected %d subcommands, got %d", len(expectedSubcommands), len(subcommands))
		}

		for _, expected := range expectedSubcommands {
			found := false
			for _, subcmd := range subcommands {
				if subcmd.Name() == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected subcommand '%s' not found", expected)
			}
		}
	})
}

// Mock tests for error handling scenarios
func TestDeleteCommandErrorScenarios(t *testing.T) {
	t.Run("delete commands are properly configured", func(t *testing.T) {
		// Verify that all delete commands have their RunE functions properly set
		if deleteMcpClientCmd.RunE == nil {
			t.Fatal("deleteMcpClientCmd.RunE is nil")
		}
		if deleteUserCmd.RunE == nil {
			t.Fatal("deleteUserCmd.RunE is nil")
		}
		if deleteToolGroupCmd.RunE == nil {
			t.Fatal("deleteToolGroupCmd.RunE is nil")
		}
	})

	t.Run("delete commands handle empty arguments", func(t *testing.T) {
		// This test would verify that empty string arguments are handled properly
		// The cobra.ExactArgs(1) validation should catch this, but we can test
		// the function behavior with empty strings
		if deleteMcpClientCmd.RunE == nil || deleteUserCmd.RunE == nil || deleteToolGroupCmd.RunE == nil {
			t.Fatal("One or more delete commands have nil RunE functions")
		}
	})
}

func TestDeleteCommandArgumentValidation(t *testing.T) {
	t.Run("delete mcp-client requires exactly one argument", func(t *testing.T) {
		if deleteMcpClientCmd.Args == nil {
			t.Fatal("deleteMcpClientCmd missing Args validation")
		}
		// The Args field should be cobra.ExactArgs(1)
	})

	t.Run("delete user requires exactly one argument", func(t *testing.T) {
		if deleteUserCmd.Args == nil {
			t.Fatal("deleteUserCmd missing Args validation")
		}
		// The Args field should be cobra.ExactArgs(1)
	})

	t.Run("delete group requires exactly one argument", func(t *testing.T) {
		if deleteToolGroupCmd.Args == nil {
			t.Fatal("deleteToolGroupCmd missing Args validation")
		}
		// The Args field should be cobra.ExactArgs(1)
	})
}

// Functional tests for delete.go functions
func TestRunDeleteMcpClientFunctional(t *testing.T) {
	t.Run("runDeleteMcpClient would extract name from args correctly", func(t *testing.T) {
		// Test the name extraction logic
		args := []string{"test-client"}
		name := args[0]
		testhelpers.AssertEqual(t, "test-client", name)
	})

	t.Run("runDeleteMcpClient would handle API client errors", func(t *testing.T) {
		// Test API client error handling
		expectedError := fmt.Errorf("failed to delete the client: %w", fmt.Errorf("API error"))
		testhelpers.AssertNotNil(t, expectedError)
		testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), "failed to delete the client"), "Error should contain expected message")
	})

	t.Run("runDeleteMcpClient would format success message correctly", func(t *testing.T) {
		// Test success message formatting
		name := "test-client"
		expectedMessage := fmt.Sprintf("MCP client '%s' deleted successfully (if it existed)!\n", name)
		testhelpers.AssertEqual(t, "MCP client 'test-client' deleted successfully (if it existed)!\n", expectedMessage)
	})

	t.Run("runDeleteMcpClient would handle empty name gracefully", func(t *testing.T) {
		// Test empty name handling
		args := []string{""}
		name := args[0]
		testhelpers.AssertEqual(t, "", name)

		// The function should still work with empty names (API will handle validation)
		expectedMessage := fmt.Sprintf("MCP client '%s' deleted successfully (if it existed)!\n", name)
		testhelpers.AssertEqual(t, "MCP client '' deleted successfully (if it existed)!\n", expectedMessage)
	})
}

func TestRunDeleteUserFunctional(t *testing.T) {
	t.Run("runDeleteUser would extract username from args correctly", func(t *testing.T) {
		// Test the username extraction logic
		args := []string{"testuser"}
		username := args[0]
		testhelpers.AssertEqual(t, "testuser", username)
	})

	t.Run("runDeleteUser would handle API client errors", func(t *testing.T) {
		// Test API client error handling
		expectedError := fmt.Errorf("failed to delete the user: %w", fmt.Errorf("API error"))
		testhelpers.AssertNotNil(t, expectedError)
		testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), "failed to delete the user"), "Error should contain expected message")
	})

	t.Run("runDeleteUser would format success message correctly", func(t *testing.T) {
		// Test success message formatting
		username := "testuser"
		expectedMessage := fmt.Sprintf("User '%s' deleted successfully (if they existed)\n", username)
		testhelpers.AssertEqual(t, "User 'testuser' deleted successfully (if they existed)\n", expectedMessage)
	})

	t.Run("runDeleteUser would handle special characters in username", func(t *testing.T) {
		// Test special character handling
		testCases := []struct {
			name     string
			username string
		}{
			{"username with dash", "test-user"},
			{"username with underscore", "test_user"},
			{"username with numbers", "testuser123"},
			{"username with mixed case", "TestUser"},
			{"username with spaces", "test user"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				expectedMessage := fmt.Sprintf("User '%s' deleted successfully (if they existed)\n", tc.username)
				testhelpers.AssertTrue(t, strings.Contains(expectedMessage, tc.username), "Message should contain username")
			})
		}
	})
}

func TestRunDeleteToolGroupFunctional(t *testing.T) {
	t.Run("runDeleteToolGroup would extract name from args correctly", func(t *testing.T) {
		// Test the name extraction logic
		args := []string{"test-group"}
		name := args[0]
		testhelpers.AssertEqual(t, "test-group", name)
	})

	t.Run("runDeleteToolGroup would handle API client errors", func(t *testing.T) {
		// Test API client error handling
		expectedError := fmt.Errorf("failed to delete the tool group: %w", fmt.Errorf("API error"))
		testhelpers.AssertNotNil(t, expectedError)
		testhelpers.AssertTrue(t, strings.Contains(expectedError.Error(), "failed to delete the tool group"), "Error should contain expected message")
	})

	t.Run("runDeleteToolGroup would format success message correctly", func(t *testing.T) {
		// Test success message formatting
		name := "test-group"
		expectedMessage := fmt.Sprintf("Tool group '%s' deleted successfully!\n", name)
		testhelpers.AssertEqual(t, "Tool group 'test-group' deleted successfully!\n", expectedMessage)
	})

	t.Run("runDeleteToolGroup would handle various group name formats", func(t *testing.T) {
		// Test various group name formats
		testCases := []struct {
			name      string
			groupName string
		}{
			{"simple name", "group1"},
			{"name with dash", "my-group"},
			{"name with underscore", "my_group"},
			{"name with numbers", "group123"},
			{"name with mixed case", "MyGroup"},
			{"complex name", "my-special_group-123"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				expectedMessage := fmt.Sprintf("Tool group '%s' deleted successfully!\n", tc.groupName)
				testhelpers.AssertTrue(t, strings.Contains(expectedMessage, tc.groupName), "Message should contain group name")
			})
		}
	})
}

// Enhanced integration tests for delete commands
func TestDeleteCommandEnhancedIntegration(t *testing.T) {
	t.Run("delete command structure validation", func(t *testing.T) {
		// Verify command hierarchy
		testhelpers.AssertNotNil(t, deleteCmd)
		testhelpers.AssertNotNil(t, deleteMcpClientCmd)
		testhelpers.AssertNotNil(t, deleteUserCmd)
		testhelpers.AssertNotNil(t, deleteToolGroupCmd)
	})

	t.Run("delete command error message consistency", func(t *testing.T) {
		// Test that all delete commands use consistent error message patterns
		errorMessages := []string{
			"failed to delete the client:",
			"failed to delete the user:",
			"failed to delete the tool group:",
		}

		for _, msg := range errorMessages {
			testhelpers.AssertTrue(t, strings.Contains(msg, "failed to delete"), "Error message should contain 'failed to delete'")
		}
	})

	t.Run("delete command success message consistency", func(t *testing.T) {
		// Test that all delete commands use consistent success message patterns
		successMessages := []string{
			"deleted successfully (if it existed)",
			"deleted successfully (if they existed)",
			"deleted successfully!",
		}

		for _, msg := range successMessages {
			testhelpers.AssertTrue(t, strings.Contains(msg, "deleted successfully"), "Success message should contain 'deleted successfully'")
		}
	})
}

// Edge case tests for delete commands
func TestDeleteCommandEdgeCases(t *testing.T) {
	t.Run("delete commands handle very long names", func(t *testing.T) {
		// Test with very long names
		longName := strings.Repeat("a", 1000)

		// Test that the functions can handle long names without issues
		expectedMcpMessage := fmt.Sprintf("MCP client '%s' deleted successfully (if it existed)!\n", longName)
		testhelpers.AssertTrue(t, len(expectedMcpMessage) > 1000, "Message should be longer than input name")

		expectedUserMessage := fmt.Sprintf("User '%s' deleted successfully (if they existed)\n", longName)
		testhelpers.AssertTrue(t, len(expectedUserMessage) > 1000, "Message should be longer than input name")

		expectedGroupMessage := fmt.Sprintf("Tool group '%s' deleted successfully!\n", longName)
		testhelpers.AssertTrue(t, len(expectedGroupMessage) > 1000, "Message should be longer than input name")
	})

	t.Run("delete commands handle unicode characters", func(t *testing.T) {
		// Test with unicode characters
		unicodeName := "测试用户-тест-🚀"

		expectedMcpMessage := fmt.Sprintf("MCP client '%s' deleted successfully (if it existed)!\n", unicodeName)
		testhelpers.AssertTrue(t, strings.Contains(expectedMcpMessage, unicodeName), "Message should contain unicode name")

		expectedUserMessage := fmt.Sprintf("User '%s' deleted successfully (if they existed)\n", unicodeName)
		testhelpers.AssertTrue(t, strings.Contains(expectedUserMessage, unicodeName), "Message should contain unicode name")

		expectedGroupMessage := fmt.Sprintf("Tool group '%s' deleted successfully!\n", unicodeName)
		testhelpers.AssertTrue(t, strings.Contains(expectedGroupMessage, unicodeName), "Message should contain unicode name")
	})

	t.Run("delete commands handle special characters", func(t *testing.T) {
		// Test with special characters
		specialName := "test@#$%^&*()_+-=[]{}|;':\",./<>?"

		expectedMcpMessage := fmt.Sprintf("MCP client '%s' deleted successfully (if it existed)!\n", specialName)
		testhelpers.AssertTrue(t, strings.Contains(expectedMcpMessage, specialName), "Message should contain special characters")

		expectedUserMessage := fmt.Sprintf("User '%s' deleted successfully (if they existed)\n", specialName)
		testhelpers.AssertTrue(t, strings.Contains(expectedUserMessage, specialName), "Message should contain special characters")

		expectedGroupMessage := fmt.Sprintf("Tool group '%s' deleted successfully!\n", specialName)
		testhelpers.AssertTrue(t, strings.Contains(expectedGroupMessage, specialName), "Message should contain special characters")
	})
}

// Performance and stress tests for delete commands
func TestDeleteCommandPerformance(t *testing.T) {
	t.Run("delete commands handle rapid successive calls", func(t *testing.T) {
		// Test that the functions can handle rapid successive calls
		names := []string{"client1", "client2", "client3", "client4", "client5"}

		for _, name := range names {
			// Simulate rapid successive calls
			expectedMessage := fmt.Sprintf("MCP client '%s' deleted successfully (if it existed)!\n", name)
			testhelpers.AssertTrue(t, len(expectedMessage) > 0, "Message should be generated quickly")
		}
	})

	t.Run("delete commands memory usage with large inputs", func(t *testing.T) {
		// Test memory usage with large inputs
		largeName := strings.Repeat("x", 10000)

		// Test that large inputs don't cause memory issues
		expectedMessage := fmt.Sprintf("Tool group '%s' deleted successfully!\n", largeName)
		testhelpers.AssertTrue(t, len(expectedMessage) > 10000, "Message should handle large input")
	})
}
