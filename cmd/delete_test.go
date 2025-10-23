package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestDeleteCommandStructure(t *testing.T) {
	t.Parallel()

	// Test command properties
	testhelpers.AssertEqual(t, "delete", deleteCmd.Use)
	testhelpers.AssertEqual(t, "Delete entities from mcpjungle", deleteCmd.Short)

	// Test command annotations
	annotationTests := []testhelpers.CommandAnnotationTest{
		{Key: "group", Expected: string(subCommandGroupAdvanced)},
		{Key: "order", Expected: "5"},
	}
	testhelpers.TestCommandAnnotations(t, deleteCmd.Annotations, annotationTests)

	// Test subcommands count
	subcommands := deleteCmd.Commands()
	testhelpers.AssertEqual(t, 3, len(subcommands))
}

func TestDeleteMcpClientSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "mcp-client [name]", deleteMcpClientCmd.Use)
	testhelpers.AssertEqual(t, "Delete an MCP client (Enterprise mode)", deleteMcpClientCmd.Short)
	testhelpers.AssertNotNil(t, deleteMcpClientCmd.Long)
	testhelpers.AssertTrue(t, len(deleteMcpClientCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, deleteMcpClientCmd.RunE)
	testhelpers.AssertNotNil(t, deleteMcpClientCmd.Args)

	// Test long description content
	longDesc := deleteMcpClientCmd.Long
	expectedPhrases := []string{
		"Delete an MCP client from the registry",
		"instantly revokes all access",
		"Enterprise mode",
	}

	for _, phrase := range expectedPhrases {
		testhelpers.AssertTrue(t, testhelpers.Contains(longDesc, phrase),
			"Expected long description to contain: "+phrase)
	}
}

func TestDeleteUserSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "user [username]", deleteUserCmd.Use)
	testhelpers.AssertEqual(t, "Delete a user (Enterprise mode)", deleteUserCmd.Short)
	testhelpers.AssertNotNil(t, deleteUserCmd.Long)
	testhelpers.AssertTrue(t, len(deleteUserCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, deleteUserCmd.RunE)
	testhelpers.AssertNotNil(t, deleteUserCmd.Args)

	// Test long description content
	longDesc := deleteUserCmd.Long
	expectedPhrases := []string{
		"Delete a user from mcpjungle",
		"instantly revokes all access",
	}

	for _, phrase := range expectedPhrases {
		testhelpers.AssertTrue(t, testhelpers.Contains(longDesc, phrase),
			"Expected long description to contain: "+phrase)
	}
}

func TestDeleteToolGroupSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "group [name]", deleteToolGroupCmd.Use)
	testhelpers.AssertEqual(t, "Delete a tool group", deleteToolGroupCmd.Short)
	testhelpers.AssertNotNil(t, deleteToolGroupCmd.Long)
	testhelpers.AssertTrue(t, len(deleteToolGroupCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, deleteToolGroupCmd.RunE)
	testhelpers.AssertNotNil(t, deleteToolGroupCmd.Args)

	// Test long description content
	longDesc := deleteToolGroupCmd.Long
	expectedPhrases := []string{
		"Delete a tool group from mcpjungle",
		"endpoint is no longer available",
		"MCP clients are relying on the endpoint",
		"only deletes the group itself",
		"Tools are only deleted when you deregister",
	}

	for _, phrase := range expectedPhrases {
		testhelpers.AssertTrue(t, testhelpers.Contains(longDesc, phrase),
			"Expected long description to contain: "+phrase)
	}
}

// Integration tests for delete commands
func TestDeleteCommandIntegration(t *testing.T) {
	// Verify that deleteCmd is properly initialized
	testhelpers.AssertNotNil(t, deleteCmd)

	// Test all delete subcommands are properly configured
	subcommands := deleteCmd.Commands()
	expectedSubcommands := []string{"mcp-client", "user", "group"}

	testhelpers.AssertEqual(t, len(expectedSubcommands), len(subcommands))

	for _, expected := range expectedSubcommands {
		found := false
		for _, subcmd := range subcommands {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		testhelpers.AssertTrue(t, found, "Expected subcommand '"+expected+"' not found")
	}
}

// Test argument validation
func TestDeleteCommandArgumentValidation(t *testing.T) {
	// Test that commands properly validate arguments
	testhelpers.AssertNotNil(t, deleteMcpClientCmd.Args)
	testhelpers.AssertNotNil(t, deleteUserCmd.Args)
	testhelpers.AssertNotNil(t, deleteToolGroupCmd.Args)
}
