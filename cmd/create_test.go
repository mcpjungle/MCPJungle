package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestCreateCommandStructure(t *testing.T) {
	t.Parallel()
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "create", createCmd.Use)
		testhelpers.AssertEqual(t, "Create resources", createCmd.Short)
	})

	t.Run("command_annotations", func(t *testing.T) {
		annotationTests := []testhelpers.CommandAnnotationTest{
			{Key: "group", Expected: string(subCommandGroupAdvanced)},
			{Key: "order", Expected: "3"},
		}
		testhelpers.TestCommandAnnotations(t, createCmd.Annotations, annotationTests)
	})

	t.Run("subcommands_count", func(t *testing.T) {
		subcommands := createCmd.Commands()
		testhelpers.AssertEqual(t, 3, len(subcommands))
	})
}

func TestCreateMcpClientSubcommand(t *testing.T) {
	t.Parallel()
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "mcp-client [name]", createMcpClientCmd.Use)
		testhelpers.AssertEqual(t, "Create an authenticated MCP client (Production mode)", createMcpClientCmd.Short)
		testhelpers.AssertNotNil(t, createMcpClientCmd.Long)
		testhelpers.AssertTrue(t, len(createMcpClientCmd.Long) > 0, "Long description should not be empty")
	})

	t.Run("command_functions", func(t *testing.T) {
		testhelpers.AssertNotNil(t, createMcpClientCmd.RunE)
		testhelpers.AssertNotNil(t, createMcpClientCmd.Args)
	})

	t.Run("command_flags", func(t *testing.T) {
		flagTests := []struct {
			name        string
			flagName    string
			description string
		}{
			{"allow_flag", "allow", "Allow flag should have usage description"},
			{"description_flag", "description", "Description flag should have usage description"},
		}

		for _, tt := range flagTests {
			t.Run(tt.name, func(t *testing.T) {
				flag := createMcpClientCmd.Flags().Lookup(tt.flagName)
				testhelpers.AssertNotNil(t, flag)
				testhelpers.AssertTrue(t, len(flag.Usage) > 0, tt.description)
			})
		}
	})
}

func TestCreateUserSubcommand(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "user [username]", createUserCmd.Use)
		testhelpers.AssertEqual(t, "Create a new user (Production mode)", createUserCmd.Short)
		testhelpers.AssertNotNil(t, createUserCmd.Long)
		testhelpers.AssertTrue(t, len(createUserCmd.Long) > 0, "Long description should not be empty")
	})

	t.Run("command_functions", func(t *testing.T) {
		testhelpers.AssertNotNil(t, createUserCmd.RunE)
		testhelpers.AssertNotNil(t, createUserCmd.Args)
	})
}

func TestCreateToolGroupSubcommand(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "group", createToolGroupCmd.Use)
		testhelpers.AssertEqual(t, "Create a Group of MCP Tools", createToolGroupCmd.Short)
		testhelpers.AssertNotNil(t, createToolGroupCmd.Long)
		testhelpers.AssertTrue(t, len(createToolGroupCmd.Long) > 0, "Long description should not be empty")
	})

	t.Run("command_functions", func(t *testing.T) {
		testhelpers.AssertNotNil(t, createToolGroupCmd.RunE)
	})

	t.Run("command_flags", func(t *testing.T) {
		confFlag := createToolGroupCmd.Flags().Lookup("conf")
		testhelpers.AssertNotNil(t, confFlag)
		testhelpers.AssertTrue(t, len(confFlag.Usage) > 0, "Conf flag should have usage description")
		// Note: The flag is marked as required in the init function with MarkFlagRequired
	})
}

func TestCreateCommandVariables(t *testing.T) {
	t.Run("command_variables_initialized", func(t *testing.T) {
		// Test that command variables are properly initialized to empty values
		testhelpers.AssertEqual(t, "", createMcpClientCmdAllowedServers)
		testhelpers.AssertEqual(t, "", createMcpClientCmdDescription)
		testhelpers.AssertEqual(t, "", createToolGroupConfigFilePath)
	})
}
