package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/client"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

func TestRunGetResource_Metadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/resource":
			_ = json.NewEncoder(w).Encode(types.Resource{
				Name:        "server__file",
				URI:         "file:///tmp/test.txt",
				MIMEType:    "text/plain",
				Description: "Sample file",
				Enabled:     true,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	origClient := apiClient
	origServer := getResourceCmdServerName
	origRead := getResourceCmdRead
	defer func() {
		apiClient = origClient
		getResourceCmdServerName = origServer
		getResourceCmdRead = origRead
	}()

	apiClient = client.NewClient(server.URL, "", http.DefaultClient)
	getResourceCmdServerName = ""
	getResourceCmdRead = false

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := runGetResource(cmd, []string{"file:///tmp/test.txt"}); err != nil {
		t.Fatalf("runGetResource returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Resource: server__file") {
		t.Fatalf("expected metadata output, got: %s", output)
	}
	if strings.Contains(output, "Content 1:") {
		t.Fatalf("did not expect read output in metadata mode, got: %s", output)
	}
}

func TestRunGetResource_Read(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/resources/read":
			_ = json.NewEncoder(w).Encode(types.ResourceReadResult{
				Contents: []map[string]any{
					{
						"uri":      "file:///tmp/test.txt",
						"mimeType": "application/json",
						"text":     "{\"hello\":\"world\"}",
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	origClient := apiClient
	origServer := getResourceCmdServerName
	origRead := getResourceCmdRead
	defer func() {
		apiClient = origClient
		getResourceCmdServerName = origServer
		getResourceCmdRead = origRead
	}()

	apiClient = client.NewClient(server.URL, "", http.DefaultClient)
	getResourceCmdServerName = ""
	getResourceCmdRead = true

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := runGetResource(cmd, []string{"file:///tmp/test.txt"}); err != nil {
		t.Fatalf("runGetResource returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Content 1:") {
		t.Fatalf("expected read output, got: %s", output)
	}
	if !strings.Contains(output, "\"hello\": \"world\"") {
		t.Fatalf("expected pretty-printed JSON output, got: %s", output)
	}
	if strings.Contains(output, "Resource: server__file") {
		t.Fatalf("did not expect metadata output in read mode, got: %s", output)
	}
}
