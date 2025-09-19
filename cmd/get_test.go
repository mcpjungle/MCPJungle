package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestGetCommandStructure(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "get", getCmd.Use)
		testhelpers.AssertEqual(t, "Get information about a specific resource", getCmd.Short)
	})

	t.Run("command_annotations", func(t *testing.T) {
		annotationTests := []testhelpers.CommandAnnotationTest{
			{Key: "group", Expected: string(subCommandGroupAdvanced)},
			{Key: "order", Expected: "7"},
		}
		testhelpers.TestCommandAnnotations(t, getCmd.Annotations, annotationTests)
	})

	t.Run("subcommands_count", func(t *testing.T) {
		subcommands := getCmd.Commands()
		testhelpers.AssertEqual(t, 1, len(subcommands))
	})
}

func TestGetGroupSubcommand(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "group [name]", getGroupCmd.Use)
		testhelpers.AssertEqual(t, "Get information about a specific Tool Group", getGroupCmd.Short)
		testhelpers.AssertNotNil(t, getGroupCmd.Long)
		testhelpers.AssertTrue(t, len(getGroupCmd.Long) > 0, "Long description should not be empty")
	})

	t.Run("command_functions", func(t *testing.T) {
		testhelpers.AssertNotNil(t, getGroupCmd.RunE)
		testhelpers.AssertNotNil(t, getGroupCmd.Args)
	})

	t.Run("long_description_content", func(t *testing.T) {
		longDesc := getGroupCmd.Long
		expectedPhrases := []string{
			"Get information about a specific Tool Group by name",
			"returns the configuration of the Tool Group",
			"which tools are included",
		}

		for _, phrase := range expectedPhrases {
			testhelpers.AssertTrue(t, testhelpers.Contains(longDesc, phrase),
				"Expected long description to contain: "+phrase)
		}
	})
}

func TestGetCommandIntegration(t *testing.T) {
	t.Run("subcommand_configuration", func(t *testing.T) {
		subcommands := getCmd.Commands()
		testhelpers.AssertEqual(t, 1, len(subcommands))
		testhelpers.AssertEqual(t, "group", subcommands[0].Name())
	})
}
