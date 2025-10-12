package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestEnableCommandStructure(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "enable", enableCmd.Use)
		testhelpers.AssertEqual(t, "Enable MCP entities like tools & prompts globally", enableCmd.Short)
		testhelpers.AssertNotNil(t, enableCmd.Long)
		testhelpers.AssertTrue(t, len(enableCmd.Long) > 0, "Long description should not be empty")
	})

	t.Run("command_annotations", func(t *testing.T) {
		annotationTests := []testhelpers.CommandAnnotationTest{
			{Key: "group", Expected: string(subCommandGroupAdvanced)},
			{Key: "order", Expected: "3"},
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
			"Enable one or more tools or prompts globally",
			"For backward-compatibility, you can still run 'enable [name]'",
		}

		for _, phrase := range expectedPhrases {
			testhelpers.AssertTrue(t, testhelpers.Contains(longDesc, phrase),
				"Expected long description to contain: "+phrase)
		}
	})
}
