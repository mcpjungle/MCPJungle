package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mcpjungle/mcpjungle/client"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

func TestRunListTools_GroupUsesEffectiveToolsAPI(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/tool-groups/test-group":
			_ = json.NewEncoder(w).Encode(types.GetToolGroupResponse{
				ToolGroup: &types.ToolGroup{Name: "test-group", Description: "desc"},
			})
		case "/api/v0/tool-groups/test-group/effective-tools":
			_ = json.NewEncoder(w).Encode(map[string]any{"tools": []string{"from_server", "ghost"}})
		case "/api/v0/tools":
			_ = json.NewEncoder(w).Encode([]*types.Tool{{Name: "from_server", Description: "ok", Enabled: true}, {Name: "other", Description: "skip", Enabled: true}})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	origClient := apiClient
	origServer := listToolsCmdServerName
	origGroup := listToolsCmdGroupName
	defer func() {
		apiClient = origClient
		listToolsCmdServerName = origServer
		listToolsCmdGroupName = origGroup
	}()

	apiClient = client.NewClient(server.URL, "", http.DefaultClient)
	listToolsCmdServerName = ""
	listToolsCmdGroupName = "test-group"

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := runListTools(cmd, nil); err != nil {
		t.Fatalf("runListTools returned error: %v", err)
	}

	output := out.String()
	if !bytes.Contains([]byte(output), []byte("from_server")) {
		t.Fatalf("expected output to contain resolved tool, got: %s", output)
	}
	if bytes.Contains([]byte(output), []byte("other")) {
		t.Fatalf("did not expect output to contain non-group tool, got: %s", output)
	}
	if bytes.Contains([]byte(output), []byte("ghost")) {
		t.Fatalf("did not expect output to contain non-existing tool, got: %s", output)
	}
}
