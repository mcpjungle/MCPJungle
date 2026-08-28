package configresolver

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestExpandEnvPlaceholders(t *testing.T) {
	t.Setenv("MCPJ_TEST_HOST", "example.com")
	t.Setenv("MCPJ_TEST_TOKEN", "secret-token")

	testCases := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name:  "exact placeholder",
			input: "${MCPJ_TEST_TOKEN}",
			want:  "secret-token",
		},
		{
			name:  "embedded placeholder",
			input: "https://${MCPJ_TEST_HOST}/mcp/${MCPJ_TEST_TOKEN}",
			want:  "https://example.com/mcp/secret-token",
		},
		{
			name:  "plain string unchanged",
			input: "no-placeholders-here",
			want:  "no-placeholders-here",
		},
		{
			name:    "missing environment variable",
			input:   "${MCPJ_TEST_MISSING}",
			wantErr: "environment variable MCPJ_TEST_MISSING is not set",
		},
		{
			name:    "invalid placeholder",
			input:   "${MCPJ_TEST_TOKEN",
			wantErr: "invalid configuration placeholder",
		},
	}
	resolver := NewResolver(nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolver.expandPlaceholders(tc.input)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestExpandSecretProviderPlaceholders(t *testing.T) {
	t.Setenv("MCPJ_TEST_HOST", "example.com")
	provider := &fakeProvider{
		values: map[string]string{
			"github-token":     "secret-token",
			"mcpjungle/nested": "nested-secret",
		},
	}
	resolver := NewResolver(map[string]SecretProvider{"keychain": provider})

	testCases := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name:  "keychain placeholder",
			input: "${keychain:github-token}",
			want:  "secret-token",
		},
		{
			name:  "opaque provider key",
			input: "${keychain:mcpjungle/nested}",
			want:  "nested-secret",
		},
		{
			name:  "multiple provider and environment placeholders",
			input: "https://${MCPJ_TEST_HOST}/mcp?first=${keychain:github-token}&second=${keychain:mcpjungle/nested}",
			want:  "https://example.com/mcp?first=secret-token&second=nested-secret",
		},
		{
			name:    "unknown provider",
			input:   "${vault:github-token}",
			wantErr: `unsupported secret provider "vault"`,
		},
		{
			name:    "empty provider key",
			input:   "${keychain:}",
			wantErr: `invalid secret reference "keychain:"`,
		},
		{
			name:    "provider error",
			input:   "${keychain:missing}",
			wantErr: `fake secret "missing" was not found`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolver.expandPlaceholders(tc.input)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestResolveConfigEnvVars(t *testing.T) {
	type nestedConfig struct {
		Command string
		Headers map[string]string
	}
	type testConfig struct {
		Name        string
		URL         string
		Args        []string
		Env         map[string]string
		Nested      nestedConfig
		AccessToken types.AccessTokenRef
	}

	t.Setenv("MCPJ_TEST_NAME", "affine-main")
	t.Setenv("MCPJ_TEST_ID", "workspace-123")
	t.Setenv("MCPJ_TEST_TOKEN", "token-abc")

	cfg := testConfig{
		Name: "server-${MCPJ_TEST_NAME}",
		URL:  "https://app.affine.pro/api/workspaces/${MCPJ_TEST_ID}/mcp",
		Args: []string{"--token=${MCPJ_TEST_TOKEN}", "--fixed=value"},
		Env: map[string]string{
			"AFFINE_TOKEN": "${MCPJ_TEST_TOKEN}",
		},
		Nested: nestedConfig{
			Command: "run-${MCPJ_TEST_NAME}",
			Headers: map[string]string{
				"Authorization": "Bearer ${MCPJ_TEST_TOKEN}",
			},
		},
		AccessToken: types.AccessTokenRef{
			Env: "TOKEN_${MCPJ_TEST_NAME}",
		},
	}

	if err := ResolveEnvVars(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "server-affine-main" {
		t.Fatalf("expected resolved name, got %q", cfg.Name)
	}
	if cfg.URL != "https://app.affine.pro/api/workspaces/workspace-123/mcp" {
		t.Fatalf("expected resolved URL, got %q", cfg.URL)
	}
	if cfg.Args[0] != "--token=token-abc" {
		t.Fatalf("expected resolved arg, got %q", cfg.Args[0])
	}
	if cfg.Env["AFFINE_TOKEN"] != "token-abc" {
		t.Fatalf("expected resolved env value, got %q", cfg.Env["AFFINE_TOKEN"])
	}
	if cfg.Nested.Command != "run-affine-main" {
		t.Fatalf("expected resolved nested command, got %q", cfg.Nested.Command)
	}
	if cfg.Nested.Headers["Authorization"] != "Bearer token-abc" {
		t.Fatalf("expected resolved header, got %q", cfg.Nested.Headers["Authorization"])
	}
	if cfg.AccessToken.Env != "TOKEN_affine-main" {
		t.Fatalf("expected resolved nested struct field, got %q", cfg.AccessToken.Env)
	}
}

func TestResolverResolvesSecretsInNestedValues(t *testing.T) {
	type nestedConfig struct {
		BearerToken string
		Headers     map[string]string
	}

	cfg := struct {
		Env    map[string]string
		Args   []string
		Nested any
	}{
		Env: map[string]string{
			"GITHUB_TOKEN": "${keychain:github-token}",
		},
		Args: []string{"--token=${keychain:github-token}"},
		Nested: nestedConfig{
			BearerToken: "${keychain:github-token}",
			Headers: map[string]string{
				"Authorization": "Bearer ${keychain:github-token}",
			},
		},
	}

	resolver := NewResolver(map[string]SecretProvider{
		"keychain": &fakeProvider{values: map[string]string{"github-token": "secret-token"}},
	})
	if err := resolver.Resolve(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Env["GITHUB_TOKEN"] != "secret-token" {
		t.Fatalf("expected resolved environment value, got %q", cfg.Env["GITHUB_TOKEN"])
	}
	if cfg.Args[0] != "--token=secret-token" {
		t.Fatalf("expected resolved argument, got %q", cfg.Args[0])
	}
	nested := cfg.Nested.(nestedConfig)
	if nested.BearerToken != "secret-token" {
		t.Fatalf("expected resolved bearer token, got %q", nested.BearerToken)
	}
	if nested.Headers["Authorization"] != "Bearer secret-token" {
		t.Fatalf("expected resolved header, got %q", nested.Headers["Authorization"])
	}
}

func TestResolveConfigEnvVarsMissingVariable(t *testing.T) {
	cfg := struct {
		URL string
	}{
		URL: "https://example.com/${MCPJ_TEST_NOT_SET}",
	}

	err := ResolveEnvVars(&cfg)
	if err == nil {
		t.Fatal("expected an error for a missing environment variable")
	}
	if !strings.Contains(err.Error(), "environment variable MCPJ_TEST_NOT_SET is not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakeProvider struct {
	values map[string]string
}

func (p *fakeProvider) Resolve(key string) (string, error) {
	value, ok := p.values[key]
	if !ok {
		return "", fmt.Errorf("fake secret %q was not found", key)
	}
	return value, nil
}
