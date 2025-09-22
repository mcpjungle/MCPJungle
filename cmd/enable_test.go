package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestEnableCommandStructure(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "enable [name]", enableCmd.Use)
		testhelpers.AssertEqual(t, "Enable one or more MCP tools globally", enableCmd.Short)
		testhelpers.AssertNotNil(t, enableCmd.Long)
		testhelpers.AssertTrue(t, len(enableCmd.Long) > 0, "Long description should not be empty")
	})

	t.Run("command_annotations", func(t *testing.T) {
		annotationTests := []testhelpers.CommandAnnotationTest{
			{Key: "group", Expected: string(subCommandGroupAdvanced)},
			{Key: "order", Expected: "2"},
		}
		testhelpers.TestCommandAnnotations(t, enableCmd.Annotations, annotationTests)
	})

	t.Run("command_functions", func(t *testing.T) {
		testhelpers.AssertNotNil(t, enableCmd.RunE)
		testhelpers.AssertNotNil(t, enableCmd.Args)
	})

	t.Run("long_description_content", func(t *testing.T) {
		longDesc := enableCmd.Long
		expectedPhrases := []string{
			"Specify the name of a tool or MCP server",
			"enable it in the mcp proxy",
			"all tools provided by that server will be enabled",
			"can be viewed and called by mcp clients",
		}

		for _, phrase := range expectedPhrases {
			testhelpers.AssertTrue(t, testhelpers.Contains(longDesc, phrase),
				"Expected long description to contain: "+phrase)
		}
	})
}
