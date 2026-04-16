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

func TestRunListResources_PrintsNamesURIsAndDescriptions(t *testing.T) {
	resourceURI := "mcpj://res/server/ZmlsZTovLy90bXAvdGVzdC50eHQ"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/resources":
			_ = json.NewEncoder(w).Encode([]*types.Resource{
				{
					Name:        "server__file",
					URI:         resourceURI,
					MIMEType:    "text/plain",
					Description: "Sample file",
					Enabled:     true,
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	origClient := apiClient
	origServer := listResourcesCmdServerName
	defer func() {
		apiClient = origClient
		listResourcesCmdServerName = origServer
	}()

	apiClient = client.NewClient(server.URL, "", http.DefaultClient)
	listResourcesCmdServerName = ""

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := runListResources(cmd, nil); err != nil {
		t.Fatalf("runListResources returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "1. server__file") {
		t.Fatalf("expected resource name in output, got: %s", output)
	}
	if !strings.Contains(output, "Sample file") {
		t.Fatalf("expected resource description in output, got: %s", output)
	}
	if !strings.Contains(output, "URI: "+resourceURI) {
		t.Fatalf("expected URI in list output, got: %s", output)
	}
	if strings.Contains(output, "MIME Type: text/plain") {
		t.Fatalf("did not expect MIME type in list output, got: %s", output)
	}
	if strings.Contains(output, "[ENABLED]") {
		t.Fatalf("did not expect status in list output, got: %s", output)
	}
}
