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
	resourceURI := "mcpj://res/server/ZmlsZTovLy90bXAvdGVzdC50eHQ"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/resources/get":
			var request types.ResourceGetRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			if request.URI != resourceURI {
				t.Fatalf("expected resource URI in request body, got: %s", request.URI)
			}
			_ = json.NewEncoder(w).Encode(&types.Resource{
				Name:        "server__file",
				URI:         resourceURI,
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
	origRead := getResourceCmdRead
	defer func() {
		apiClient = origClient
		getResourceCmdRead = origRead
	}()

	apiClient = client.NewClient(server.URL, "", http.DefaultClient)
	getResourceCmdRead = false

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := runGetResource(cmd, []string{resourceURI}); err != nil {
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
	resourceURI := "mcpj://res/server/ZmlsZTovLy90bXAvdGVzdC50eHQ"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/resources/get":
			var request types.ResourceGetRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			if request.URI != resourceURI {
				t.Fatalf("expected resource URI in request body, got: %s", request.URI)
			}
			_ = json.NewEncoder(w).Encode(&types.Resource{
				Name:        "server__file",
				URI:         resourceURI,
				MIMEType:    "application/json",
				Description: "Sample file",
				Enabled:     true,
			})
		case "/api/v0/resources/read":
			var request types.ResourceReadRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			if request.URI != resourceURI {
				t.Fatalf("expected requested URI in read request, got: %s", request.URI)
			}

			_ = json.NewEncoder(w).Encode(types.ResourceReadResult{
				Contents: []map[string]any{
					{
						"uri":      resourceURI,
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
	origRead := getResourceCmdRead
	defer func() {
		apiClient = origClient
		getResourceCmdRead = origRead
	}()

	apiClient = client.NewClient(server.URL, "", http.DefaultClient)
	getResourceCmdRead = true

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := runGetResource(cmd, []string{resourceURI}); err != nil {
		t.Fatalf("runGetResource returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Content 1:") {
		t.Fatalf("expected read output, got: %s", output)
	}
	if !strings.Contains(output, "\"hello\": \"world\"") {
		t.Fatalf("expected pretty-printed JSON output, got: %s", output)
	}
	if !strings.Contains(output, "Resource: server__file") {
		t.Fatalf("expected resolved resource name in read mode, got: %s", output)
	}
	if !strings.Contains(output, "URI: "+resourceURI) {
		t.Fatalf("expected resolved URI in read mode, got: %s", output)
	}
}
