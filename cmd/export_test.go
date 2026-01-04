package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveTargetDirForExport(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() (string, error)
		cleanup       func(string)
		expectedError bool
		validateDir   func(t *testing.T, dir string)
	}{
		{
			name: "default target directory",
			setup: func() (string, error) {
				exportCmdTargetDir = ""
				tmpDir := filepath.Join(t.TempDir(), "default_test")
				return tmpDir, nil
			},
			cleanup: func(dir string) {
				os.RemoveAll(dir)
			},
			expectedError: false,
			validateDir: func(t *testing.T, dir string) {
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					t.Errorf("expected directory to be created, but it doesn't exist")
				}
			},
		},
		{
			name: "absolute path",
			setup: func() (string, error) {
				dir := filepath.Join(t.TempDir(), "absolute_test")
				exportCmdTargetDir = dir
				return dir, nil
			},
			cleanup: func(dir string) {
				_ = os.RemoveAll(dir)
			},
			expectedError: false,
			validateDir: func(t *testing.T, dir string) {
				if !filepath.IsAbs(dir) {
					t.Errorf("expected absolute path, got %s", dir)
				}
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					t.Errorf("expected directory to be created")
				}
			},
		},
		{
			name: "relative path",
			setup: func() (string, error) {
				tmpDir := t.TempDir()
				_ = os.Chdir(tmpDir)
				exportCmdTargetDir = "relative_test"
				return filepath.Join(tmpDir, "relative_test"), nil
			},
			cleanup: func(dir string) {
				_ = os.RemoveAll(dir)
			},
			expectedError: false,
			validateDir: func(t *testing.T, dir string) {
				if !filepath.IsAbs(dir) {
					t.Errorf("expected absolute path after resolution, got %s", dir)
				}
			},
		},
		{
			name: "tilde expansion at home",
			setup: func() (string, error) {
				home, _ := os.UserHomeDir()
				testDir := filepath.Join(home, ".mcpjungle_test_"+strings.ReplaceAll(t.Name(), "/", "_"))
				exportCmdTargetDir = "~/.mcpjungle_test_" + strings.ReplaceAll(t.Name(), "/", "_")
				return testDir, nil
			},
			cleanup: func(dir string) {
				_ = os.RemoveAll(dir)
			},
			expectedError: false,
			validateDir: func(t *testing.T, dir string) {
				home, _ := os.UserHomeDir()
				if !strings.HasPrefix(dir, home) {
					t.Errorf("expected path to be under home directory, got %s", dir)
				}
			},
		},
		{
			name: "tilde as single argument",
			setup: func() (string, error) {
				home, _ := os.UserHomeDir()
				exportCmdTargetDir = "~"
				return home, nil
			},
			cleanup: func(dir string) {
				// Don't delete home directory
			},
			expectedError: true, // Home directory likely not empty
			validateDir:   func(t *testing.T, dir string) {},
		},
		{
			name: "nested directory creation",
			setup: func() (string, error) {
				tmpDir := t.TempDir()
				nestedDir := filepath.Join(tmpDir, "level1", "level2", "level3")
				exportCmdTargetDir = nestedDir
				return nestedDir, nil
			},
			cleanup: func(dir string) {
				_ = os.RemoveAll(filepath.Dir(filepath.Dir(filepath.Dir(dir))))
			},
			expectedError: false,
			validateDir: func(t *testing.T, dir string) {
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					t.Errorf("expected nested directory to be created")
				}
			},
		},
		{
			name: "directory already exists but empty",
			setup: func() (string, error) {
				tmpDir := t.TempDir()
				exportCmdTargetDir = tmpDir
				return tmpDir, nil
			},
			cleanup: func(dir string) {
				_ = os.RemoveAll(dir)
			},
			expectedError: false,
			validateDir: func(t *testing.T, dir string) {
				entries, _ := os.ReadDir(dir)
				if len(entries) != 0 {
					t.Errorf("expected empty directory")
				}
			},
		},
		{
			name: "directory not empty",
			setup: func() (string, error) {
				tmpDir := t.TempDir()
				exportCmdTargetDir = tmpDir
				// Create a file in the directory
				_ = os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0o644)
				return tmpDir, nil
			},
			cleanup: func(dir string) {
				_ = os.RemoveAll(dir)
			},
			expectedError: true,
			validateDir:   func(t *testing.T, dir string) {},
		},
		{
			name: "invalid permissions for directory creation",
			setup: func() (string, error) {
				tmpDir := t.TempDir()
				restrictedDir := filepath.Join(tmpDir, "restricted")
				_ = os.Mkdir(restrictedDir, 0o000)
				exportCmdTargetDir = filepath.Join(restrictedDir, "subdir")
				return exportCmdTargetDir, nil
			},
			cleanup: func(dir string) {
				_ = os.Chmod(filepath.Dir(dir), 0o755)
				_ = os.RemoveAll(filepath.Dir(filepath.Dir(dir)))
			},
			expectedError: true,
			validateDir:   func(t *testing.T, dir string) {},
		},
		{
			name: "path with dots and slashes",
			setup: func() (string, error) {
				tmpDir := t.TempDir()
				exportCmdTargetDir = filepath.Join(tmpDir, "test", "..", "test", ".", "export")
				return filepath.Join(tmpDir, "test", "export"), nil
			},
			cleanup: func(dir string) {
				_ = os.RemoveAll(filepath.Dir(filepath.Dir(dir)))
			},
			expectedError: false,
			validateDir: func(t *testing.T, dir string) {
				// Verify the path has been cleaned
				if strings.Contains(dir, "..") || strings.Contains(dir, "/.") {
					t.Errorf("expected path to be cleaned, got %s", dir)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedDir, _ := tt.setup()
			defer tt.cleanup(expectedDir)

			result, err := resolveTargetDirForExport()

			if (err != nil) != tt.expectedError {
				t.Errorf("expected error: %v, got error: %v", tt.expectedError, err != nil)
			}

			if !tt.expectedError && err == nil {
				tt.validateDir(t, result)
			}
		})
	}
}
