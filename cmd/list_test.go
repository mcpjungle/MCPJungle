package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestListCommandStructure(t *testing.T) {
	t.Parallel()

	// Test command properties
	testhelpers.AssertEqual(t, "list", listCmd.Use)
	testhelpers.AssertEqual(t, "List resources like MCP servers, tools, etc", listCmd.Short)

	// Test command annotations
	annotationTests := []testhelpers.CommandAnnotationTest{
		{Key: "group", Expected: string(subCommandGroupBasic)},
		{Key: "order", Expected: "3"},
	}
	testhelpers.TestCommandAnnotations(t, listCmd.Annotations, annotationTests)

	// Test subcommands count
	subcommands := listCmd.Commands()
	testhelpers.AssertEqual(t, 5, len(subcommands))
}

func TestListToolsSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "tools", listToolsCmd.Use)
	testhelpers.AssertEqual(t, "List available tools", listToolsCmd.Short)
	testhelpers.AssertNotNil(t, listToolsCmd.Long)
	testhelpers.AssertTrue(t, len(listToolsCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, listToolsCmd.RunE)

	// Test command flags
	serverFlag := listToolsCmd.Flags().Lookup("server")
	testhelpers.AssertNotNil(t, serverFlag)
	testhelpers.AssertTrue(t, len(serverFlag.Usage) > 0, "Server flag should have usage description")
}

func TestListServersSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "servers", listServersCmd.Use)
	testhelpers.AssertEqual(t, "List registered MCP servers", listServersCmd.Short)

	// Test command functions
	testhelpers.AssertNotNil(t, listServersCmd.RunE)
}

func TestListMcpClientsSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "mcp-clients", listMcpClientsCmd.Use)
	testhelpers.AssertEqual(t, "List MCP clients (Production mode)", listMcpClientsCmd.Short)
	testhelpers.AssertNotNil(t, listMcpClientsCmd.Long)
	testhelpers.AssertTrue(t, len(listMcpClientsCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, listMcpClientsCmd.RunE)
}

func TestListUsersSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "users", listUsersCmd.Use)
	testhelpers.AssertEqual(t, "List users (Production mode)", listUsersCmd.Short)
	testhelpers.AssertNotNil(t, listUsersCmd.Long)
	testhelpers.AssertTrue(t, len(listUsersCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, listUsersCmd.RunE)
}

func TestListGroupsSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "groups", listGroupsCmd.Use)
	testhelpers.AssertEqual(t, "List tool groups", listGroupsCmd.Short)

	// Test command functions
	testhelpers.AssertNotNil(t, listGroupsCmd.RunE)
}

// Integration tests for list commands
func TestListCommandIntegration(t *testing.T) {
	// Verify that listCmd is properly initialized
	testhelpers.AssertNotNil(t, listCmd)

	// Test all list subcommands are properly configured
	subcommands := listCmd.Commands()
	expectedSubcommands := []string{"tools", "servers", "mcp-clients", "users", "groups"}

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
