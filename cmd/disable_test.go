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
