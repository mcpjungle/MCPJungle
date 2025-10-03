package cmd

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func TestInvokeCommandStructure(t *testing.T) {
	t.Parallel()

	// Test command properties
	testhelpers.AssertEqual(t, "invoke <name>", invokeToolCmd.Use)
	testhelpers.AssertEqual(t, "Invoke a tool", invokeToolCmd.Short)
	testhelpers.AssertNotNil(t, invokeToolCmd.Long)
	testhelpers.AssertTrue(t, len(invokeToolCmd.Long) > 0, "Long description should not be empty")

	// Test command annotations
	annotationTests := []testhelpers.CommandAnnotationTest{
		{Key: "group", Expected: string(subCommandGroupBasic)},
		{Key: "order", Expected: "5"},
	}
	testhelpers.TestCommandAnnotations(t, invokeToolCmd.Annotations, annotationTests)

	// Test command functions
	testhelpers.AssertNotNil(t, invokeToolCmd.RunE)
	testhelpers.AssertNotNil(t, invokeToolCmd.Args)

	// Test command flags
	inputFlag := invokeToolCmd.Flags().Lookup("input")
	testhelpers.AssertNotNil(t, inputFlag)
	testhelpers.AssertTrue(t, len(inputFlag.Usage) > 0, "Input flag should have usage description")

	groupFlag := invokeToolCmd.Flags().Lookup("group")
	testhelpers.AssertNotNil(t, groupFlag)
	testhelpers.AssertTrue(t, len(groupFlag.Usage) > 0, "Group flag should have usage description")

	// Test long description content
	longDesc := invokeToolCmd.Long
	expectedPhrases := []string{
		"Invokes a tool supplied by a registered MCP server",
	}

	for _, phrase := range expectedPhrases {
		testhelpers.AssertTrue(t, testhelpers.Contains(longDesc, phrase),
			"Expected long description to contain: "+phrase)
	}
}

// Integration tests for invoke command
func TestInvokeCommandIntegration(t *testing.T) {
	// Verify that invokeToolCmd is properly initialized
	testhelpers.AssertNotNil(t, invokeToolCmd)
}

func TestHandleResourceContent(t *testing.T) {
	tests := []struct {
		name           string
		input          map[string]any
		expectedOutput string
		expectedError  string
		expectFile     bool
		expectedExt    string
	}{
		{
			name: "text resource content",
			input: map[string]any{
				"resource": map[string]any{
					"uri":      "file://test.txt",
					"mimeType": "text/plain",
					"text":     "Hello, World!",
				},
			},
			expectedOutput: "Resource URI: file://test.txt\nMIME Type: text/plain\nText Content:\nHello, World!\n",
			expectedError:  "",
			expectFile:     false,
		},
		{
			name: "text resource without mime type",
			input: map[string]any{
				"resource": map[string]any{
					"uri":  "file://test.txt",
					"text": "Hello, World!",
				},
			},
			expectedOutput: "Resource URI: file://test.txt\nText Content:\nHello, World!\n",
			expectedError:  "",
			expectFile:     false,
		},
		{
			name: "blob resource content - PDF",
			input: map[string]any{
				"resource": map[string]any{
					"uri":      "file://test.pdf",
					"mimeType": "application/pdf",
					"blob":     base64.StdEncoding.EncodeToString([]byte("fake PDF content")),
				},
			},
			expectedOutput: "Resource URI: file://test.pdf\nMIME Type: application/pdf\n",
			expectedError:  "",
			expectFile:     true,
			expectedExt:    ".pdf",
		},
		{
			name: "blob resource content - JSON",
			input: map[string]any{
				"resource": map[string]any{
					"uri":      "file://data.json",
					"mimeType": "application/json",
					"blob":     base64.StdEncoding.EncodeToString([]byte(`{"key": "value"}`)),
				},
			},
			expectedOutput: "Resource URI: file://data.json\nMIME Type: application/json\n",
			expectedError:  "",
			expectFile:     true,
			expectedExt:    ".json",
		},
		{
			name: "blob resource content - unknown MIME type",
			input: map[string]any{
				"resource": map[string]any{
					"uri":      "file://unknown.bin",
					"mimeType": "application/unknown",
					"blob":     base64.StdEncoding.EncodeToString([]byte("binary data")),
				},
			},
			expectedOutput: "Resource URI: file://unknown.bin\nMIME Type: application/unknown\n",
			expectedError:  "",
			expectFile:     true,
			expectedExt:    ".bin",
		},
		{
			name: "blob resource without MIME type",
			input: map[string]any{
				"resource": map[string]any{
					"uri":  "file://data.bin",
					"blob": base64.StdEncoding.EncodeToString([]byte("binary data")),
				},
			},
			expectedOutput: "Resource URI: file://data.bin\n",
			expectedError:  "",
			expectFile:     true,
			expectedExt:    ".bin",
		},
		{
			name: "invalid base64 blob",
			input: map[string]any{
				"resource": map[string]any{
					"uri":      "file://invalid.bin",
					"mimeType": "application/octet-stream",
					"blob":     "invalid-base64!@#$",
				},
			},
			expectedOutput: "Resource URI: file://invalid.bin\nMIME Type: application/octet-stream\n",
			expectedError:  "failed to decode base64 blob data",
			expectFile:     false,
		},
		{
			name: "missing resource field",
			input: map[string]any{
				"type": "resource",
			},
			expectedOutput: "",
			expectedError:  "resource content item does not have a valid 'resource' field",
			expectFile:     false,
		},
		{
			name: "resource without text or blob",
			input: map[string]any{
				"resource": map[string]any{
					"uri":      "file://incomplete.bin",
					"mimeType": "application/octet-stream",
				},
			},
			expectedOutput: "Resource URI: file://incomplete.bin\nMIME Type: application/octet-stream\n",
			expectedError:  "resource content does not contain 'text' or 'blob' field",
			expectFile:     false,
		},
		{
			name: "empty URI",
			input: map[string]any{
				"resource": map[string]any{
					"uri":  "",
					"text": "Hello",
				},
			},
			expectedOutput: "Resource URI: \nText Content:\nHello\n",
			expectedError:  "",
			expectFile:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}

			// Create an in-memory filesystem for each test
			fs := afero.NewMemMapFs()
			tmpDir := "/tmp"

			// Create the tmp directory in the in-memory filesystem
			err := fs.MkdirAll(tmpDir, 0o755)
			if err != nil {
				t.Fatalf("Failed to create tmp dir in memory: %v", err)
			}

			// Create a buffer to capture output, and set it to the command
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			err = unpackResourceContent(cmd, tt.input, tmpDir, fs)

			// Check error expectations
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, but got %q", tt.expectedError, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			// Check output expectations
			actualOutput := output.String()
			if !strings.Contains(actualOutput, tt.expectedOutput) {
				t.Errorf("Expected output to contain %q, but got %q", tt.expectedOutput, actualOutput)
			}

			// Check file expectations
			if tt.expectFile && err == nil {
				files, err := afero.ReadDir(fs, tmpDir)
				if err != nil {
					t.Fatalf("Failed to read temp dir: %v", err)
				}

				if len(files) == 0 {
					t.Errorf("Expected a file to be created, but none found")
				} else {
					filename := files[0].Name()
					if tt.expectedExt != "" && !strings.HasSuffix(filename, tt.expectedExt) {
						t.Errorf("Expected file with extension %q, but got %q", tt.expectedExt, filename)
					}

					// Verify output mentions the saved file
					if !strings.Contains(actualOutput, "[Resource saved as "+filename+"]") {
						t.Errorf("Expected output to mention saved file %q", filename)
					}
				}
			} else if !tt.expectFile {
				files, err := afero.ReadDir(fs, tmpDir)
				if err != nil {
					t.Fatalf("Failed to read temp dir: %v", err)
				}

				if len(files) > 0 {
					t.Errorf("Expected no files to be created, but found %d", len(files))
				}
			}
		})
	}
}
