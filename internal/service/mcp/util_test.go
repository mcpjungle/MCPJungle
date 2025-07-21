package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
)

func TestValidateServerName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "server_1", false},
		{"valid hyphen", "server-2", false},
		{"invalid slash", "server/3", true},
		{"invalid special char", "server$", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateServerName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestMergeServerToolNames(t *testing.T) {
	tests := []struct {
		server string
		tool   string
		want   string
	}{
		{"myserver", "mytool", "myserver/mytool"},
		{"myserver", "my/tool", "myserver/my/tool"},
	}
	for _, tt := range tests {
		t.Run(tt.server+"_"+tt.tool, func(t *testing.T) {
			got := mergeServerToolNames(tt.server, tt.tool)
			if got != tt.want {
				t.Errorf("mergeServerToolNames(%q, %q) = %q, want %q", tt.server, tt.tool, got, tt.want)
			}
		})
	}
}

func TestSplitServerToolName(t *testing.T) {
	tests := []struct {
		input      string
		wantServer string
		wantTool   string
		wantOK     bool
	}{
		{"server/tool", "server", "tool", true},
		{"a/b/c", "a", "b/c", true},
		{"no_separator", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			server, tool, ok := splitServerToolName(tt.input)
			if server != tt.wantServer || tool != tt.wantTool || ok != tt.wantOK {
				t.Errorf("splitServerToolName(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.input, server, tool, ok, tt.wantServer, tt.wantTool, tt.wantOK)
			}
		})
	}
}

func TestCreateMcpServerConnBlocksRedirects(t *testing.T) {
	// Test that the HTTP client blocks redirects
	// This protects against SSRF and credential leakage

	// Create a test server that returns redirects
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			// Return a redirect response
			http.Redirect(w, r, "http://evil.com/steal-credentials", http.StatusFound)
			return
		}
		// For any other path, return a simple response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer redirectServer.Close()

	// Create a test MCP server that points to the redirect endpoint
	testServer := &model.McpServer{
		URL:         redirectServer.URL + "/redirect",
		BearerToken: "test-token", // This would be leaked if redirects weren't blocked
	}

	// Attempt to create connection (this should fail due to redirect being blocked)
	ctx := context.Background()
	_, err := createMcpServerConn(ctx, testServer)

	// We expect this to fail because:
	// 1. The redirect is blocked (returns http.ErrUseLastResponse)
	// 2. The MCP initialization will fail on the redirect response
	if err == nil {
		t.Fatal("Expected error when connecting to server that returns redirects, but got none")
	}

	// The error should indicate connection/initialization failure, not a successful redirect
	errMsg := err.Error()
	if strings.Contains(errMsg, "evil.com") {
		t.Fatalf("Error message contains redirected URL, suggesting redirects were not blocked: %s", errMsg)
	}

	// This is a positive test - we WANT the connection to fail when redirects are encountered
	t.Logf("Successfully blocked redirect attempt - error: %v", err)
}

func TestCreateMcpServerConnWorksWithoutRedirects(t *testing.T) {
	// Test that normal connections still work when no redirects are involved

	// Create a test server that returns a proper MCP response
	normalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple response that won't cause MCP initialization to succeed
		// (but won't be blocked by redirect protection)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-1,"message":"test"},"id":null}`))
	}))
	defer normalServer.Close()

	// Create a test MCP server with normal URL
	testServer := &model.McpServer{
		URL: normalServer.URL,
	}

	// Attempt to create connection
	ctx := context.Background()
	_, err := createMcpServerConn(ctx, testServer)

	// We expect this to fail due to MCP initialization issues, but NOT due to redirect blocking
	if err == nil {
		t.Log("Connection succeeded (unexpected but not a redirect protection issue)")
		return
	}

	// The error should be related to MCP initialization, not redirect blocking
	errMsg := err.Error()
	if strings.Contains(errMsg, "redirect") || strings.Contains(errMsg, "ErrUseLastResponse") {
		t.Fatalf("Error suggests redirect blocking interfered with normal connection: %s", errMsg)
	}

	t.Logf("Connection failed as expected for non-redirect reasons: %v", err)
}
