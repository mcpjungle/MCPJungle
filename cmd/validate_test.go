package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestValidateCommandStructure(t *testing.T) {
	t.Parallel()

	if validateCmd.Use != "validate <file>" {
		t.Fatalf("expected validate command Use to be %q, got %q", "validate <file>", validateCmd.Use)
	}
	if validateCmd.Short != "Validate an MCPJungle JSON config file" {
		t.Fatalf("unexpected validate command Short: %q", validateCmd.Short)
	}
	if validateCmd.RunE == nil {
		t.Fatal("validate command missing RunE")
	}
	if validateCmd.Annotations["group"] != string(subCommandGroupBasic) {
		t.Fatalf("expected validate command in basic group, got %q", validateCmd.Annotations["group"])
	}
}

func TestRunValidateConfigPrintsDetectedType(t *testing.T) {
	t.Parallel()

	file := writeValidateTestConfig(t, `{
		"name": "context7",
		"transport": "streamable_http",
		"url": "https://mcp.context7.com/mcp"
	}`)

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := runValidateConfig(cmd, []string{file}); err != nil {
		t.Fatalf("runValidateConfig returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "valid MCPJungle config") {
		t.Fatalf("expected success output, got: %s", output)
	}
	if !strings.Contains(output, "mcp-server") {
		t.Fatalf("expected detected config type in output, got: %s", output)
	}
}

func writeValidateTestConfig(t *testing.T, contents string) string {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "validate-config-*.json")
	if err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}
	if _, err := file.WriteString(contents); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("failed to close temp config: %v", err)
	}
	return file.Name()
}
