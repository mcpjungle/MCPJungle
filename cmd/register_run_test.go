package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/client"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

func TestRunRegisterMCPServer_PrintsPromptsAndResourcesWithoutTools(t *testing.T) {
	resourceURI := "mcpj://res/server/ZmlsZTovLy90bXAvdGVzdC50eHQ"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/servers":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(types.McpServer{
				Name:      "test-server",
				Transport: string(types.TransportStreamableHTTP),
			})
		case "/api/v0/tools":
			_ = json.NewEncoder(w).Encode([]*types.Tool{})
		case "/api/v0/prompts":
			_ = json.NewEncoder(w).Encode([]model.Prompt{
				{Name: "summarize", Description: "Summarize content"},
			})
		case "/api/v0/resources":
			_ = json.NewEncoder(w).Encode([]*types.Resource{
				{
					Name:        "server__file",
					URI:         resourceURI,
					MIMEType:    "text/plain",
					Description: "Sample file",
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	origClient := apiClient
	origName := registerCmdServerName
	origURL := registerCmdServerURL
	origDesc := registerCmdServerDesc
	origToken := registerCmdBearerToken
	origConfig := registerCmdServerConfigFilePath
	origForce := registerCmdForce
	defer func() {
		apiClient = origClient
		registerCmdServerName = origName
		registerCmdServerURL = origURL
		registerCmdServerDesc = origDesc
		registerCmdBearerToken = origToken
		registerCmdServerConfigFilePath = origConfig
		registerCmdForce = origForce
	}()

	apiClient = client.NewClient(server.URL, "", http.DefaultClient)
	registerCmdServerName = "test-server"
	registerCmdServerURL = "http://upstream.example/mcp"
	registerCmdServerDesc = ""
	registerCmdBearerToken = ""
	registerCmdServerConfigFilePath = ""
	registerCmdForce = false

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := runRegisterMCPServer(cmd, nil); err != nil {
		t.Fatalf("runRegisterMCPServer returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "The following prompts are now available from this server:") {
		t.Fatalf("expected prompts section, got: %s", output)
	}
	if !strings.Contains(output, "The following resources are now available from this server:") {
		t.Fatalf("expected resources section, got: %s", output)
	}
	if !strings.Contains(output, "1. server__file") {
		t.Fatalf("expected resource name in output, got: %s", output)
	}
	if !strings.Contains(output, "URI: "+resourceURI) {
		t.Fatalf("expected resource URI in register output, got: %s", output)
	}
	if strings.Contains(output, "This server does not provide any tools.") {
		t.Fatalf("did not expect early no-tools message, got: %s", output)
	}
}

func TestRunRegisterMCPServer_PrintsEmptySummaryWhenNoCapabilitiesExist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/servers":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(types.McpServer{
				Name:      "empty-server",
				Transport: string(types.TransportStreamableHTTP),
			})
		case "/api/v0/tools":
			_ = json.NewEncoder(w).Encode([]*types.Tool{})
		case "/api/v0/prompts":
			_ = json.NewEncoder(w).Encode([]model.Prompt{})
		case "/api/v0/resources":
			_ = json.NewEncoder(w).Encode([]*types.Resource{})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	origClient := apiClient
	origName := registerCmdServerName
	origURL := registerCmdServerURL
	origDesc := registerCmdServerDesc
	origToken := registerCmdBearerToken
	origConfig := registerCmdServerConfigFilePath
	origForce := registerCmdForce
	defer func() {
		apiClient = origClient
		registerCmdServerName = origName
		registerCmdServerURL = origURL
		registerCmdServerDesc = origDesc
		registerCmdBearerToken = origToken
		registerCmdServerConfigFilePath = origConfig
		registerCmdForce = origForce
	}()

	apiClient = client.NewClient(server.URL, "", http.DefaultClient)
	registerCmdServerName = "empty-server"
	registerCmdServerURL = "http://upstream.example/mcp"
	registerCmdServerDesc = ""
	registerCmdBearerToken = ""
	registerCmdServerConfigFilePath = ""
	registerCmdForce = false

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := runRegisterMCPServer(cmd, nil); err != nil {
		t.Fatalf("runRegisterMCPServer returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "This server does not provide any tools, prompts or resources.") {
		t.Fatalf("expected empty summary, got: %s", output)
	}
}
