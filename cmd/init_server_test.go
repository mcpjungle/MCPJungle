package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestInitServerCommandStructure(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "init-server", initServerCmd.Use)
		testhelpers.AssertEqual(t, "Initialize the MCPJungle Server (for Enterprise Mode only)", initServerCmd.Short)
		testhelpers.AssertNotNil(t, initServerCmd.Long)
		testhelpers.AssertTrue(t, len(initServerCmd.Long) > 0, "Long description should not be empty")
	})

	t.Run("command_annotations", func(t *testing.T) {
		testhelpers.AssertNotNil(t, initServerCmd.Annotations)

		annotationTests := []struct {
			key      string
			expected string
		}{
			{"group", string(subCommandGroupAdvanced)},
			{"order", "5"},
		}

		for _, tt := range annotationTests {
			t.Run(tt.key, func(t *testing.T) {
				value, exists := initServerCmd.Annotations[tt.key]
				testhelpers.AssertTrue(t, exists, "Missing '"+tt.key+"' annotation")
				testhelpers.AssertEqual(t, tt.expected, value)
			})
		}
	})

	t.Run("command_has_run_function", func(t *testing.T) {
		testhelpers.AssertNotNil(t, initServerCmd.RunE)
	})

	t.Run("long_description_content", func(t *testing.T) {
		expectedPhrases := []string{
			"If the MCPJungle Server was started in Enterprise Mode",
			"use this command to initialize the server",
			"Initialization is required before you can use the server",
		}

		for i, phrase := range expectedPhrases {
			t.Run(testhelpers.FormatError("phrase", i), func(t *testing.T) {
				testhelpers.AssertTrue(t, testhelpers.Contains(initServerCmd.Long, phrase),
					"Long description should contain: "+phrase)
			})
		}
	})
}
