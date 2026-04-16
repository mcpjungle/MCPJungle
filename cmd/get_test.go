package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestGetCommandStructure(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "get", getCmd.Use)
		testhelpers.AssertEqual(t, "Get entities like Prompts and Tool Groups", getCmd.Short)
	})

	t.Run("command_annotations", func(t *testing.T) {
		annotationTests := []testhelpers.CommandAnnotationTest{
			{Key: "group", Expected: string(subCommandGroupAdvanced)},
			{Key: "order", Expected: "1"},
		}
		testhelpers.TestCommandAnnotations(t, getCmd.Annotations, annotationTests)
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

func TestPrettifyGetPromptError(t *testing.T) {
	t.Run("drops duplicated failed-to-get-prompt prefix", func(t *testing.T) {
		// Mirrors the real wrapping chain from cmd/get.go + api/mcp_prompts.go +
		// service/mcp/prompt.go: three stacked "failed to get prompt:" prefixes.
		in := fmt.Errorf("failed to get prompt: failed to get prompt: failed to get prompt Dataset Details from MCP server hf: something broke")
		got := prettifyGetPromptError("hf__Dataset Details", in).Error()
		want := "failed to get prompt Dataset Details from MCP server hf: something broke"
		if got != want {
			t.Fatalf("prettifyGetPromptError mismatch\n got:  %q\n want: %q", got, want)
		}
	})

	t.Run("appends --arg usage hint on MCP -32602", func(t *testing.T) {
		// Real MCP server payload returned for a missing required argument.
		in := fmt.Errorf("failed to get prompt: failed to get prompt Dataset Details from MCP server hf: invalid params: MCP error -32602: Invalid arguments for prompt Dataset Details: []")
		got := prettifyGetPromptError("hf__Dataset Details", in).Error()
		if strings.Contains(got, "failed to get prompt: failed to get prompt") {
			t.Fatalf("prefix should have been collapsed, got: %q", got)
		}
		if !strings.Contains(got, "Hint:") || !strings.Contains(got, "--arg key=value") {
			t.Fatalf("expected --arg usage hint, got: %q", got)
		}
	})

	t.Run("passes non-MCP errors through unchanged (minus prefix)", func(t *testing.T) {
		in := fmt.Errorf("failed to get prompt: request failed with status: 404, message: not found")
		got := prettifyGetPromptError("hf__Missing", in).Error()
		want := "request failed with status: 404, message: not found"
		if got != want {
			t.Fatalf("unexpected result\n got:  %q\n want: %q", got, want)
		}
	})
}

