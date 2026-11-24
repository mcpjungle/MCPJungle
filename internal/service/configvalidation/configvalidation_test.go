package configvalidation

import (
	"os"
	"strings"
	"testing"
)

func TestValidateFileDetectsMCPServerConfig(t *testing.T) {
	t.Parallel()

	file := writeTempConfig(t, `{
		"name": "context7",
		"transport": "streamable_http",
		"url": "https://mcp.context7.com/mcp"
	}`)

	result, err := ValidateFile(file)
	if err != nil {
		t.Fatalf("ValidateFile returned error: %v", err)
	}
	if result.Type != ConfigTypeMCPServer {
		t.Fatalf("expected config type %q, got %q", ConfigTypeMCPServer, result.Type)
	}
}

func TestValidateFileRejectsInvalidMCPServerURL(t *testing.T) {
	t.Parallel()

	file := writeTempConfig(t, `{
		"name": "bad-server",
		"transport": "streamable_http",
		"url": "not-a-url"
	}`)

	_, err := ValidateFile(file)
	if err == nil {
		t.Fatal("expected invalid URL error")
	}
	if !strings.Contains(err.Error(), "valid http or https url") {
		t.Fatalf("expected URL validation error, got: %v", err)
	}
}

func TestValidateBytesDetectsToolGroupConfig(t *testing.T) {
	t.Parallel()

	result, err := ValidateBytes([]byte(`{
		"name": "coding-tools",
		"included_servers": ["github"],
		"excluded_tools": ["github__delete_repository"]
	}`))
	if err != nil {
		t.Fatalf("ValidateBytes returned error: %v", err)
	}
	if result.Type != ConfigTypeToolGroup {
		t.Fatalf("expected config type %q, got %q", ConfigTypeToolGroup, result.Type)
	}
}

func TestValidateBytesDetectsMCPClientConfig(t *testing.T) {
	t.Parallel()

	result, err := ValidateBytes([]byte(`{
		"name": "cursor-local",
		"allowed_servers": ["context7"],
		"access_token": "client-token"
	}`))
	if err != nil {
		t.Fatalf("ValidateBytes returned error: %v", err)
	}
	if result.Type != ConfigTypeMCPClient {
		t.Fatalf("expected config type %q, got %q", ConfigTypeMCPClient, result.Type)
	}
}

func TestValidateBytesDetectsUserConfig(t *testing.T) {
	t.Parallel()

	result, err := ValidateBytes([]byte(`{
		"name": "alice",
		"access_token": "alice-token"
	}`))
	if err != nil {
		t.Fatalf("ValidateBytes returned error: %v", err)
	}
	if result.Type != ConfigTypeUser {
		t.Fatalf("expected config type %q, got %q", ConfigTypeUser, result.Type)
	}
}

func TestValidateBytesRejectsUnknownConfig(t *testing.T) {
	t.Parallel()

	_, err := ValidateBytes([]byte(`{"name": "incomplete"}`))
	if err == nil {
		t.Fatal("expected unknown config error")
	}
	if !strings.Contains(err.Error(), "does not match any supported MCPJungle config type") {
		t.Fatalf("expected unknown config error, got: %v", err)
	}
}

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "mcpjungle-config-*.json")
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
