package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartCommandStructure(t *testing.T) {
	t.Run("start command has correct properties", func(t *testing.T) {
		if startServerCmd.Use != "start" {
			t.Errorf("Expected start command Use to be 'start', got %s", startServerCmd.Use)
		}
		if startServerCmd.Short != "Start the MCPJungle server" {
			t.Errorf("Expected start command Short to be 'Start the MCPJungle server', got %s", startServerCmd.Short)
		}
	})

	t.Run("start command has correct annotations", func(t *testing.T) {
		if startServerCmd.Annotations == nil {
			t.Fatal("Start command missing annotations")
		}

		group, hasGroup := startServerCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Start command missing 'group' annotation")
		}
		if group != string(subCommandGroupBasic) {
			t.Errorf("Expected start command group to be 'basic', got %s", group)
		}

		order, hasOrder := startServerCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Start command missing 'order' annotation")
		}
		if order != "1" {
			t.Errorf("Expected start command order to be '1', got %s", order)
		}
	})
}

func TestStartCommandFlags(t *testing.T) {
	t.Run("start command has port flag", func(t *testing.T) {
		portFlag := startServerCmd.Flags().Lookup("port")
		if portFlag == nil {
			t.Fatal("Start command missing 'port' flag")
		}
		if portFlag.Usage == "" {
			t.Error("Port flag should have usage description")
		}
	})

	t.Run("start command has enterprise flag", func(t *testing.T) {
		enterpriseFlag := startServerCmd.Flags().Lookup("enterprise")
		prodFlag := startServerCmd.Flags().Lookup("enterprise")
		if enterpriseFlag == nil && prodFlag == nil {
			t.Fatal("Start command missing 'enterprise' flag")
		}
		if enterpriseFlag.Usage == "" && prodFlag.Usage == "" {
			t.Error("enterprise flag should have usage description")
		}
	})
}

// Helper to set and unset env vars for a test
func withEnv(env map[string]string, fn func()) {
	originals := make(map[string]string)
	for k, v := range env {
		originals[k] = os.Getenv(k)
		os.Setenv(k, v)
	}
	fn()
	for k, v := range originals {
		os.Setenv(k, v)
	}
}

// Helper to create a temp file with content
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	f := filepath.Join(tmp, "val")
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return f
}

func TestGetPostgresDSN(t *testing.T) {
	baseEnv := map[string]string{
		PostgresHostEnvVar:     "localhost",
		PostgresPortEnvVar:     "5433",
		PostgresUserEnvVar:     "user",
		PostgresPasswordEnvVar: "pass",
		PostgresDBEnvVar:       "mydb",
	}

	t.Run("returns false if POSTGRES_HOST is not set", func(t *testing.T) {
		withEnv(map[string]string{
			PostgresHostEnvVar: "",
		}, func() {
			dsn, ok, err := getPostgresDSN()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok {
				t.Errorf("expected ok=false, got true")
			}
			if dsn != "" {
				t.Errorf("expected empty dsn, got %q", dsn)
			}
		})
	})

	t.Run("uses all env vars", func(t *testing.T) {
		withEnv(baseEnv, func() {
			dsn, ok, err := getPostgresDSN()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ok {
				t.Errorf("expected ok=true, got false")
			}
			want := "postgres://user:pass@localhost:5433/mydb"
			if dsn != want {
				t.Errorf("expected dsn %q, got %q", want, dsn)
			}
		})
	})

	t.Run("uses defaults for missing optional vars", func(t *testing.T) {
		withEnv(map[string]string{
			PostgresHostEnvVar: "host",
		}, func() {
			dsn, ok, err := getPostgresDSN()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ok {
				t.Errorf("expected ok=true, got false")
			}
			want := "postgres://postgres:@host:5432/postgres"
			if dsn != want {
				t.Errorf("expected dsn %q, got %q", want, dsn)
			}
		})
	})

	t.Run("uses _FILE env for DB, user, password", func(t *testing.T) {
		dbFile := writeTempFile(t, "filedb")
		userFile := writeTempFile(t, "fileuser")
		passFile := writeTempFile(t, "filepass")
		withEnv(map[string]string{
			PostgresHostEnvVar:               "host",
			PostgresDBEnvVar + "_FILE":       dbFile,
			PostgresUserEnvVar + "_FILE":     userFile,
			PostgresPasswordEnvVar + "_FILE": passFile,
		}, func() {
			dsn, ok, err := getPostgresDSN()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ok {
				t.Errorf("expected ok=true, got false")
			}
			want := "postgres://fileuser:filepass@host:5432/filedb"
			if dsn != want {
				t.Errorf("expected dsn %q, got %q", want, dsn)
			}
		})
	})

	t.Run("env var takes precedence over _FILE", func(t *testing.T) {
		dbFile := writeTempFile(t, "filedb")
		withEnv(map[string]string{
			PostgresHostEnvVar:         "host",
			PostgresDBEnvVar:           "envdb",
			PostgresDBEnvVar + "_FILE": dbFile,
		}, func() {
			dsn, ok, err := getPostgresDSN()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ok {
				t.Errorf("expected ok=true, got false")
			}
			want := "postgres://postgres:@host:5432/envdb"
			if dsn != want {
				t.Errorf("expected dsn %q, got %q", want, dsn)
			}
		})
	})

	t.Run("returns error if _FILE cannot be read", func(t *testing.T) {
		withEnv(map[string]string{
			PostgresHostEnvVar:         "host",
			PostgresDBEnvVar + "_FILE": "/nonexistent/file",
		}, func() {
			_, ok, err := getPostgresDSN()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if ok {
				t.Errorf("expected ok=false, got true")
			}
		})
	})

	t.Run("trims whitespace from _FILE values", func(t *testing.T) {
		dbFile := writeTempFile(t, "  dbwithspace \n")
		withEnv(map[string]string{
			PostgresHostEnvVar:         "host",
			PostgresDBEnvVar + "_FILE": dbFile,
		}, func() {
			dsn, ok, err := getPostgresDSN()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ok {
				t.Errorf("expected ok=true, got false")
			}
			want := "postgres://postgres:@host:5432/dbwithspace"
			if dsn != want {
				t.Errorf("expected dsn %q, got %q", want, dsn)
			}
		})
	})

	t.Run("empty password is allowed", func(t *testing.T) {
		withEnv(map[string]string{
			PostgresHostEnvVar: "host",
			PostgresUserEnvVar: "user",
		}, func() {
			dsn, ok, err := getPostgresDSN()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ok {
				t.Errorf("expected ok=true, got false")
			}
			want := "postgres://user:@host:5432/postgres"
			if dsn != want {
				t.Errorf("expected dsn %q, got %q", want, dsn)
			}
		})
	})
}

func TestGetMcpServerInitReqTimeout(t *testing.T) {
	t.Run("returns default when unset or empty", func(t *testing.T) {
		withEnv(map[string]string{
			McpServerInitReqTimeoutSecEnvVar: "",
		}, func() {
			v, err := getMcpServerInitReqTimeout()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v != McpServerInitRequestTimeoutSecondsDefault {
				t.Fatalf("expected default %d, got %d", McpServerInitRequestTimeoutSecondsDefault, v)
			}
		})
	})

	t.Run("parses valid integer value", func(t *testing.T) {
		withEnv(map[string]string{
			McpServerInitReqTimeoutSecEnvVar: "5",
		}, func() {
			v, err := getMcpServerInitReqTimeout()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v != 5 {
				t.Fatalf("expected 5, got %d", v)
			}
		})
	})

	t.Run("trims whitespace before parsing", func(t *testing.T) {
		withEnv(map[string]string{
			McpServerInitReqTimeoutSecEnvVar: "  10 \n",
		}, func() {
			v, err := getMcpServerInitReqTimeout()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v != 10 {
				t.Fatalf("expected 10, got %d", v)
			}
		})
	})

	t.Run("returns error for non-integer", func(t *testing.T) {
		withEnv(map[string]string{
			McpServerInitReqTimeoutSecEnvVar: "abc",
		}, func() {
			_, err := getMcpServerInitReqTimeout()
			if err == nil {
				t.Fatal("expected error for non-integer value, got nil")
			}
		})
	})

	t.Run("returns error for values less than 1", func(t *testing.T) {
		cases := []string{"0", "-1"}
		for _, c := range cases {
			withEnv(map[string]string{
				McpServerInitReqTimeoutSecEnvVar: c,
			}, func() {
				_, err := getMcpServerInitReqTimeout()
				if err == nil {
					t.Fatalf("expected error for value %q, got nil", c)
				}
			})
		}
	})
}
