package cmd

import (
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestReadCommandStructure(t *testing.T) {
	testhelpers.AssertEqual(t, "read", readCmd.Use)
	testhelpers.AssertEqual(t, "Read live content from MCP resources", readCmd.Short)
	testhelpers.AssertNotNil(t, readCmd.Annotations)

	testhelpers.AssertEqual(t, "resource [uri]", readResourceCmd.Use)
	testhelpers.AssertEqual(t, "Read live resource content", readResourceCmd.Short)
	testhelpers.AssertNotNil(t, readResourceCmd.Long)
	testhelpers.AssertNotNil(t, readResourceCmd.RunE)
	testhelpers.AssertNotNil(t, readResourceCmd.Args)

	serverFlag := readResourceCmd.Flags().Lookup("server")
	testhelpers.AssertNotNil(t, serverFlag)
	testhelpers.AssertTrue(t, len(serverFlag.Usage) > 0, "Server flag should have usage description")
}
