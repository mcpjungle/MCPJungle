package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestDisableCommandStructure(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "disable [name]", disableCmd.Use)
		testhelpers.AssertEqual(t, "Disable one or more MCP tools globally", disableCmd.Short)
		testhelpers.AssertNotNil(t, disableCmd.Long)
		testhelpers.AssertTrue(t, len(disableCmd.Long) > 0, "Long description should not be empty")
	})

	t.Run("command_annotations", func(t *testing.T) {
		annotationTests := []testhelpers.CommandAnnotationTest{
			{Key: "group", Expected: string(subCommandGroupAdvanced)},
			{Key: "order", Expected: "1"},
		}
		testhelpers.TestCommandAnnotations(t, disableCmd.Annotations, annotationTests)
	})

	t.Run("command_functions", func(t *testing.T) {
		testhelpers.AssertNotNil(t, disableCmd.RunE)
		testhelpers.AssertNotNil(t, disableCmd.Args)
	})

	t.Run("long_description_content", func(t *testing.T) {
		longDesc := disableCmd.Long
		expectedPhrases := []string{
			"Specify the name of a tool or MCP server",
			"disable it in the mcp proxy",
			"all tools provided by that server will be disabled",
			"cannot be viewed or called by mcp clients",
		}

		for _, phrase := range expectedPhrases {
			testhelpers.AssertTrue(t, testhelpers.Contains(longDesc, phrase),
				"Expected long description to contain: "+phrase)
		}
	})
}

func TestRunDisableTools(t *testing.T) {
	t.Run("function_definition", func(t *testing.T) {
		// Verify that the function is properly assigned to the command
		testhelpers.AssertNotNil(t, disableCmd.RunE)
	})
}

func TestDisableCommandIntegration(t *testing.T) {
	t.Run("command_integration", func(t *testing.T) {
		// Verify that disableCmd is properly initialized
		testhelpers.AssertNotNil(t, disableCmd)
	})
}

func TestDisableCommandArgumentValidation(t *testing.T) {
	t.Run("argument_validation", func(t *testing.T) {
		testhelpers.AssertNotNil(t, disableCmd.Args)
		// The Args field should be cobra.ExactArgs(1)
	})
}
