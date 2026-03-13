package accesstoken

import (
	"os"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestResolve(t *testing.T) {
	t.Run("inline token wins", func(t *testing.T) {
		token, err := Resolve(Input{Inline: "direct-token"})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if token != "direct-token" {
			t.Fatalf("expected direct-token, got: %s", token)
		}
	})

	t.Run("env token is used", func(t *testing.T) {
		env := "MCPJ_TEST_TOKEN_ENV"
		_ = os.Setenv(env, "  env-token  ")
		defer os.Unsetenv(env)

		token, err := Resolve(Input{Ref: types.AccessTokenRef{Env: env}})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if token != "env-token" {
			t.Fatalf("expected env-token, got: %s", token)
		}
	})

	t.Run("file token is used", func(t *testing.T) {
		f, err := os.CreateTemp("", "mcpj-token-*")
		if err != nil {
			t.Fatalf("create temp file: %v", err)
		}
		_ = os.WriteFile(f.Name(), []byte("  file-token\n"), 0o600)
		defer os.Remove(f.Name())

		token, err := Resolve(Input{Ref: types.AccessTokenRef{File: f.Name()}})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if token != "file-token" {
			t.Fatalf("expected file-token, got: %s", token)
		}
	})
}
