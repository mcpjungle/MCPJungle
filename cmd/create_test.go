package cmd

import (
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestCreateCommandStructure(t *testing.T) {
	t.Parallel()

	// Test command properties
	testhelpers.AssertEqual(t, "create", createCmd.Use)
	testhelpers.AssertEqual(t, "Create entities in mcpjungle", createCmd.Short)

	// Test command annotations
	annotationTests := []testhelpers.CommandAnnotationTest{
		{Key: "group", Expected: string(subCommandGroupAdvanced)},
		{Key: "order", Expected: "4"},
	}
	testhelpers.TestCommandAnnotations(t, createCmd.Annotations, annotationTests)

	// Test subcommands count
	subcommands := createCmd.Commands()
	testhelpers.AssertEqual(t, 3, len(subcommands))
}

func TestCreateMcpClientSubcommand(t *testing.T) {
	t.Parallel()

	// Test command properties
	testhelpers.AssertEqual(t, "mcp-client [name]", createMcpClientCmd.Use)
	testhelpers.AssertEqual(t, "Create an authenticated MCP client (Enterprise mode)", createMcpClientCmd.Short)
	testhelpers.AssertNotNil(t, createMcpClientCmd.Long)
	testhelpers.AssertTrue(t, len(createMcpClientCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, createMcpClientCmd.RunE)
	testhelpers.AssertNotNil(t, createMcpClientCmd.Args)

	// Test command flags
	allowFlag := createMcpClientCmd.Flags().Lookup("allow")
	testhelpers.AssertNotNil(t, allowFlag)
	testhelpers.AssertTrue(t, len(allowFlag.Usage) > 0, "Allow flag should have usage description")

	descriptionFlag := createMcpClientCmd.Flags().Lookup("description")
	testhelpers.AssertNotNil(t, descriptionFlag)
	testhelpers.AssertTrue(t, len(descriptionFlag.Usage) > 0, "Description flag should have usage description")
}

func TestCreateUserSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "user [username]", createUserCmd.Use)
	testhelpers.AssertEqual(t, "Create a new user (Enterprise mode)", createUserCmd.Short)
	testhelpers.AssertNotNil(t, createUserCmd.Long)
	testhelpers.AssertTrue(t, len(createUserCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, createUserCmd.RunE)
	testhelpers.AssertNotNil(t, createUserCmd.Args)
}

func TestCreateToolGroupSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "group", createToolGroupCmd.Use)
	testhelpers.AssertEqual(t, "Create a Group of MCP Tools", createToolGroupCmd.Short)
	testhelpers.AssertNotNil(t, createToolGroupCmd.Long)
	testhelpers.AssertTrue(t, len(createToolGroupCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, createToolGroupCmd.RunE)

	// Test command flags
	confFlag := createToolGroupCmd.Flags().Lookup("conf")
	testhelpers.AssertNotNil(t, confFlag)
	testhelpers.AssertTrue(t, len(confFlag.Usage) > 0, "Conf flag should have usage description")
}

func TestCreateCommandVariables(t *testing.T) {
	// Test that command variables are properly initialized to empty values
	testhelpers.AssertEqual(t, "", createMcpClientCmdAllowedServers)
	testhelpers.AssertEqual(t, "", createMcpClientCmdDescription)
	testhelpers.AssertEqual(t, "", createToolGroupConfigFilePath)
}

// Test allow list parsing logic
func TestParseAllowList(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", []string{}},
		{"single server", "server1", []string{"server1"}},
		{"multiple servers", "server1,server2,server3", []string{"server1", "server2", "server3"}},
		{"servers with spaces", "server1, server2 , server3", []string{"server1", "server2", "server3"}},
		{"servers with empty elements", "server1,,server2", []string{"server1", "server2"}},
		{"servers with only spaces", "server1,  ,server2", []string{"server1", "server2"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse allow list (simulating the logic from create.go)
			allowList := make([]string, 0)
			for _, s := range strings.Split(tc.input, ",") {
				trimmed := strings.TrimSpace(s)
				if trimmed != "" {
					allowList = append(allowList, trimmed)
				}
			}

			// Compare results
			if len(tc.expected) != len(allowList) {
				t.Errorf("Expected length %d, got %d", len(tc.expected), len(allowList))
				return
			}
			for i, expected := range tc.expected {
				if expected != allowList[i] {
					t.Errorf("Expected[%d] = %s, got %s", i, expected, allowList[i])
				}
			}
		})
	}
}

// Integration tests for create commands
func TestCreateCommandIntegration(t *testing.T) {
	// Verify that createCmd is properly added to rootCmd
	testhelpers.AssertNotNil(t, createCmd)

	// Test all create subcommands are properly configured
	subcommands := createCmd.Commands()
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
func TestCreateCommandArgumentValidation(t *testing.T) {
	// Test that commands properly validate arguments
	testhelpers.AssertNotNil(t, createMcpClientCmd.Args)
	testhelpers.AssertNotNil(t, createUserCmd.Args)
	// createToolGroupCmd doesn't have Args validation, which is correct

	// Test various invalid input scenarios
	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{"empty args", []string{}, true},
		{"too many args", []string{"arg1", "arg2", "arg3"}, true},
		{"valid single arg", []string{"valid-arg"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test mcp-client command args validation
			if createMcpClientCmd.Args != nil {
				err := createMcpClientCmd.Args(createMcpClientCmd, tc.args)
				if tc.expectError {
					testhelpers.AssertError(t, err)
				} else {
					testhelpers.AssertNoError(t, err)
				}
			}

			// Test user command args validation
			if createUserCmd.Args != nil {
				err := createUserCmd.Args(createUserCmd, tc.args)
				if tc.expectError {
					testhelpers.AssertError(t, err)
				} else {
					testhelpers.AssertNoError(t, err)
				}
			}
		})
	}
}
