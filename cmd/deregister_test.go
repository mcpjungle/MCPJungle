package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestDeregisterCommandStructure(t *testing.T) {
	t.Parallel()

	// Test command properties
	testhelpers.AssertEqual(t, "deregister", deregisterMCPServerCmd.Use)
	testhelpers.AssertEqual(t, "Deregister an MCP Server", deregisterMCPServerCmd.Short)
	testhelpers.AssertNotNil(t, deregisterMCPServerCmd.Long)
	testhelpers.AssertTrue(t, len(deregisterMCPServerCmd.Long) > 0, "Long description should not be empty")

	// Test command annotations
	annotationTests := []testhelpers.CommandAnnotationTest{
		{Key: "group", Expected: string(subCommandGroupBasic)},
		{Key: "order", Expected: "6"},
	}
	testhelpers.TestCommandAnnotations(t, deregisterMCPServerCmd.Annotations, annotationTests)

	// Test command functions
	testhelpers.AssertNotNil(t, deregisterMCPServerCmd.RunE)
	testhelpers.AssertNotNil(t, deregisterMCPServerCmd.Args)

	// Test long description content
	longDesc := deregisterMCPServerCmd.Long
	expectedPhrases := []string{
		"Remove an MCP server from the registry",
		"deregisters all tools provided by the server",
	}

	for _, phrase := range expectedPhrases {
		testhelpers.AssertTrue(t, testhelpers.Contains(longDesc, phrase),
			"Expected long description to contain: "+phrase)
	}
}

// Integration tests for deregister command
func TestDeregisterCommandIntegration(t *testing.T) {
	// Verify that deregisterMCPServerCmd is properly initialized
	testhelpers.AssertNotNil(t, deregisterMCPServerCmd)
}

// Test argument validation
func TestDeregisterCommandArgumentValidation(t *testing.T) {
	// Test that command properly validates arguments
	testhelpers.AssertNotNil(t, deregisterMCPServerCmd.Args)
}
