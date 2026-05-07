package e2e_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------
// Enterprise mode – authentication & RBAC
// -----------------------------------------------------------------------

func TestE2E_EnterpriseMode_Unauthenticated_Returns401(t *testing.T) {
	env := setupE2EServer(t, model.ModeEnterprise)

	endpoints := []struct{ method, path string }{
		{http.MethodGet, "/api/v0/tools"},
		{http.MethodGet, "/api/v0/prompts"},
		{http.MethodGet, "/api/v0/servers"},
	}
	for _, ep := range endpoints {
		t.Run(fmt.Sprintf("%s %s", ep.method, ep.path), func(t *testing.T) {
			resp := env.do(t, ep.method, ep.path, nil, "")
			defer drain(resp)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func TestE2E_EnterpriseMode_RegularUser_CannotWrite(t *testing.T) {
	env := setupE2EServer(t, model.ModeEnterprise)

	writeOps := []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/api/v0/servers", map[string]any{"name": "x", "transport": "stdio", "command": "echo"}},
		{http.MethodPost, "/api/v0/tool-groups", map[string]any{"name": "g"}},
		{http.MethodPost, "/api/v0/clients", map[string]any{"name": "c"}},
		{http.MethodPost, "/api/v0/users", map[string]any{"username": "u"}},
	}
	for _, op := range writeOps {
		t.Run(fmt.Sprintf("%s %s", op.method, op.path), func(t *testing.T) {
			resp := env.do(t, op.method, op.path, op.body, env.userToken)
			defer drain(resp)
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}

// -----------------------------------------------------------------------
// Enterprise mode – admin manages MCP clients (enterprise-only)
// -----------------------------------------------------------------------

func TestE2E_EnterpriseMode_AdminManagesClients(t *testing.T) {
	env := setupE2EServer(t, model.ModeEnterprise)

	resp := env.do(t, http.MethodPost, "/api/v0/clients",
		map[string]any{"name": "myapp", "allow_list": []string{"*"}}, env.adminToken)
	defer drain(resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created map[string]any
	decodeJSON(t, resp, &created)
	assert.Equal(t, "myapp", created["name"])
	assert.NotEmpty(t, created["access_token"])

	resp = env.do(t, http.MethodGet, "/api/v0/clients", nil, env.adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	drain(resp)
}

// -----------------------------------------------------------------------
// Enterprise mode – MCP proxy: client token required and ACL filtering
// -----------------------------------------------------------------------

// TestE2E_EnterpriseMode_McpProxy_RequiresClientToken verifies that only a
// valid MCP client token (not a user/admin token) grants access to /mcp.
func TestE2E_EnterpriseMode_McpProxy_RequiresClientToken(t *testing.T) {
	env := setupE2EServer(t, model.ModeEnterprise)

	for _, token := range []string{"", env.userToken, env.adminToken} {
		resp := env.do(t, http.MethodGet, "/mcp", nil, token)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		drain(resp)
	}

	clientToken := createMcpClient(t, env, "auth-client", []string{"*"})
	resp := env.do(t, http.MethodGet, "/mcp", nil, clientToken)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode, "valid client token must not return 401")
	drain(resp)
}

// TestE2E_EnterpriseMode_McpProxy_AllowList_ListAndInvoke registers two servers
// (svc-a and svc-b) and creates a client restricted to svc-a only.
//
// Tools:
//   - ListTools returns only svc-a tools, not svc-b tools
//   - CallTool on an allowed tool succeeds; on a restricted tool returns IsError=true
//
// Prompts:
//   - ListPrompts returns prompts from ALL servers regardless of allow list
//     (ACL filtering is not implemented for prompt listing)
//   - GetPrompt on an allowed prompt succeeds; on a restricted prompt returns an error
func TestE2E_EnterpriseMode_McpProxy_AllowList_ListAndInvoke(t *testing.T) {
	env := setupE2EServer(t, model.ModeEnterprise)

	// Register two independent server instances so we can test cross-server scoping.
	registerEverythingServerAs(t, env, "svc-a", env.adminToken)
	registerEverythingServerAs(t, env, "svc-b", env.adminToken)

	// Client is restricted to svc-a only.
	c := newMCPProxyClient(t, env, createMcpClient(t, env, "scoped-client", []string{"svc-a"}))

	// --- Tools ---

	t.Run("list tools: only allowed server's tools visible", func(t *testing.T) {
		result, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
		require.NoError(t, err)

		names := make([]string, 0, len(result.Tools))
		for _, tool := range result.Tools {
			names = append(names, tool.Name)
		}
		assert.Contains(t, names, "svc-a__echo", "allowed server tool must be visible")
		assert.NotContains(t, names, "svc-b__echo", "restricted server tool must not be visible")
	})

	t.Run("invoke allowed tool succeeds", func(t *testing.T) {
		result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "svc-a__echo",
				Arguments: map[string]any{"message": "hello from svc-a"},
			},
		})
		require.NoError(t, err)
		require.False(t, result.IsError)
		first, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		assert.Contains(t, first.Text, "hello from svc-a")
	})

	t.Run("invoke restricted tool returns error", func(t *testing.T) {
		// The ACL check returns a Go error (not an MCP IsError result), so the
		// mcp-go framework surfaces it as a protocol-level error on the client.
		_, err := c.CallTool(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "svc-b__echo",
				Arguments: map[string]any{"message": "should be blocked"},
			},
		})
		assert.Error(t, err, "calling a tool from a restricted server must return an error")
	})

	// --- Prompts ---

	t.Run("list prompts: not filtered by allow list (ACL for prompt listing not implemented)", func(t *testing.T) {
		result, err := c.ListPrompts(context.Background(), mcp.ListPromptsRequest{})
		require.NoError(t, err)

		names := make([]string, 0, len(result.Prompts))
		for _, p := range result.Prompts {
			names = append(names, p.Name)
		}
		// Both servers' prompts appear regardless of the allow list.
		assert.Contains(t, names, "svc-a__simple-prompt")
		assert.Contains(t, names, "svc-b__simple-prompt")
	})

	t.Run("get allowed prompt succeeds", func(t *testing.T) {
		result, err := c.GetPrompt(context.Background(), mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{Name: "svc-a__simple-prompt"},
		})
		require.NoError(t, err)
		require.NotEmpty(t, result.Messages)
	})

	t.Run("get restricted prompt returns error", func(t *testing.T) {
		_, err := c.GetPrompt(context.Background(), mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{Name: "svc-b__simple-prompt"},
		})
		assert.Error(t, err, "fetching a prompt from a restricted server must return an error")
	})
}

func TestE2E_EnterpriseMode_McpProxy_StripsInboundHeadersForUpstreamCalls(t *testing.T) {
	env := setupE2EServer(t, model.ModeEnterprise)

	upstream := mcpserver.NewMCPServer("header-check", "0.1.0")
	upstream.AddTool(
		mcp.NewTool("echo", mcp.WithString("message", mcp.Required())),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, _ := req.GetArguments()["message"].(string)
			return mcp.NewToolResultText(msg), nil
		},
	)
	upstream.AddPrompt(
		mcp.Prompt{Name: "simple-prompt"},
		func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return &mcp.GetPromptResult{
				Messages: []mcp.PromptMessage{
					mcp.NewPromptMessage(mcp.RoleAssistant, mcp.TextContent{Type: "text", Text: "prompt ok"}),
				},
			}, nil
		},
	)
	upstream.AddResource(
		mcp.Resource{URI: "resource://header-check/status", Name: "status", MIMEType: "text/plain"},
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      req.Params.URI,
					MIMEType: "text/plain",
					Text:     "resource ok",
				},
			}, nil
		},
	)

	upstreamHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-Test-Forward") != "" {
			http.Error(w, "downstream headers leaked upstream", http.StatusUnauthorized)
			return
		}
		mcpserver.NewStreamableHTTPServer(upstream).ServeHTTP(w, r)
	}))
	defer upstreamHTTP.Close()

	resp := env.do(t, http.MethodPost, "/api/v0/servers", map[string]any{
		"name":        "header-proxy",
		"description": "Header scrub regression server",
		"transport":   "streamable_http",
		"url":         upstreamHTTP.URL,
	}, env.adminToken)
	defer drain(resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	clientToken := createMcpClient(t, env, "header-proxy-client", []string{"header-proxy"})
	c, err := client.NewStreamableHttpClient(env.baseURL+"/mcp", transport.WithHTTPHeaders(map[string]string{
		"Authorization":  "Bearer " + clientToken,
		"X-Test-Forward": "downstream-custom-header",
	}))
	require.NoError(t, err)
	defer c.Close()

	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "e2e-header-test-client",
				Version: "1.0.0",
			},
		},
	})
	require.NoError(t, err)

	toolRes, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "header-proxy__echo",
			Arguments: map[string]any{"message": "tool ok"},
		},
	})
	require.NoError(t, err)
	require.False(t, toolRes.IsError)

	promptRes, err := c.GetPrompt(context.Background(), mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{Name: "header-proxy__simple-prompt"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, promptRes.Messages)

	resourceURI := "mcpj://res/header-proxy/" + base64.RawStdEncoding.EncodeToString([]byte("resource://header-check/status"))
	resourceRes, err := c.ReadResource(context.Background(), mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: resourceURI},
	})
	require.NoError(t, err)
	require.NotEmpty(t, resourceRes.Contents)
}
