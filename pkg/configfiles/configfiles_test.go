package configfiles

import (
	"os"
	"path/filepath"
	"testing"
)

type testConfig struct {
	Name string `json:"name"`
}

func TestReadJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "entity.json")

	if err := os.WriteFile(path, []byte(`{"name":"alpha"}`), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := ReadJSONFile[testConfig](path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got.Name != "alpha" {
		t.Fatalf("expected name alpha, got %s", got.Name)
	}
}

func TestReadJSONFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "entity.json")

	if err := os.WriteFile(path, []byte(`{"name":`), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := ReadJSONFile[testConfig](path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoadDesired(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte(`{"name":"a"}`), 0o644); err != nil {
		t.Fatalf("write file a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.json"), []byte(`{"name":"b"}`), 0o644); err != nil {
		t.Fatalf("write file b: %v", err)
	}

	desired, blocked, errs := LoadDesired[testConfig](dir, func(c testConfig) string { return c.Name })
	if len(errs) != 0 {
		t.Fatalf("expected no parse errors, got: %v", errs)
	}
	if len(blocked) != 0 {
		t.Fatalf("expected no blocked files, got: %v", blocked)
	}
	if len(desired) != 2 {
		t.Fatalf("expected 2 desired entries, got %d", len(desired))
	}
	if desired["a"].Name != "a" || desired["a"].Path == "" || desired["a"].Hash == "" {
		t.Fatalf("expected metadata for a to be populated, got: %+v", desired["a"])
	}
}

func TestLoadDesired_DuplicateName(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte(`{"name":"dupe"}`), 0o644); err != nil {
		t.Fatalf("write file a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.json"), []byte(`{"name":"dupe"}`), 0o644); err != nil {
		t.Fatalf("write file b: %v", err)
	}

	desired, blocked, errs := LoadDesired[testConfig](dir, func(c testConfig) string { return c.Name })
	if len(errs) == 0 {
		t.Fatal("expected duplicate-name error, got none")
	}
	if len(desired) != 1 {
		t.Fatalf("expected 1 desired entry after duplicate conflict, got %d", len(desired))
	}
	if len(blocked) != 2 {
		t.Fatalf("expected 2 blocked files for duplicate conflict, got %d", len(blocked))
	}
}
