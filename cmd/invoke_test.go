package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestInvokeCommandStructure(t *testing.T) {
	t.Parallel()

	// Test command properties
	testhelpers.AssertEqual(t, "invoke <name>", invokeToolCmd.Use)
	testhelpers.AssertEqual(t, "Invoke a tool", invokeToolCmd.Short)
	testhelpers.AssertNotNil(t, invokeToolCmd.Long)
	testhelpers.AssertTrue(t, len(invokeToolCmd.Long) > 0, "Long description should not be empty")

	// Test command annotations
	annotationTests := []testhelpers.CommandAnnotationTest{
		{Key: "group", Expected: string(subCommandGroupBasic)},
		{Key: "order", Expected: "5"},
	}
	testhelpers.TestCommandAnnotations(t, invokeToolCmd.Annotations, annotationTests)

	// Test command functions
	testhelpers.AssertNotNil(t, invokeToolCmd.RunE)
	testhelpers.AssertNotNil(t, invokeToolCmd.Args)

	// Test command flags
	inputFlag := invokeToolCmd.Flags().Lookup("input")
	testhelpers.AssertNotNil(t, inputFlag)
	testhelpers.AssertTrue(t, len(inputFlag.Usage) > 0, "Input flag should have usage description")

	// Test long description content
	longDesc := invokeToolCmd.Long
	expectedPhrases := []string{
		"Invokes a tool supplied by a registered MCP server",
	}

	for _, phrase := range expectedPhrases {
		testhelpers.AssertTrue(t, testhelpers.Contains(longDesc, phrase),
			"Expected long description to contain: "+phrase)
	}
}

// Integration tests for invoke command
func TestInvokeCommandIntegration(t *testing.T) {
	// Verify that invokeToolCmd is properly initialized
	testhelpers.AssertNotNil(t, invokeToolCmd)
}
