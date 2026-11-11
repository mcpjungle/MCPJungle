package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newLazyMCPClient connects to an MCP endpoint with ?mode=lazy appended.
func newLazyMCPClient(t *testing.T, env *e2eEnv, path string, token string) *client.Client {
	t.Helper()
	opts := []transport.StreamableHTTPCOption{}
	if token != "" {
		opts = append(opts, transport.WithHTTPHeaders(map[string]string{
			"Authorization": "Bearer " + token,
		}))
	}
	c, err := client.NewStreamableHttpClient(env.baseURL+path+"?mode=lazy", opts...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "e2e-lazy-client", Version: "1.0.0"},
		},
	})
	require.NoError(t, err)
	return c
}

func listToolNames(t *testing.T, c *client.Client) []string {
	t.Helper()
	res, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	names := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	return names
}

func callTool(t *testing.T, c *client.Client, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	res, err := c.CallTool(context.Background(), req)
	require.NoError(t, err)
	return res
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok, "expected text content")
	return text.Text
}

func TestE2E_LazyMode_DiscoveryActivationAndInvoke(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	c := newLazyMCPClient(t, env, "/mcp", "")

	// Before activation only the meta-tools are exposed.
	assert.ElementsMatch(t,
		[]string{"mcpjungle__list_servers", "mcpjungle__show_server_tools", "mcpjungle__invoke"},
		listToolNames(t, c),
	)

	listRes := callTool(t, c, "mcpjungle__list_servers", nil)
	require.False(t, listRes.IsError, resultText(t, listRes))
	var servers []map[string]any
	require.NoError(t, json.Unmarshal([]byte(resultText(t, listRes)), &servers))
	require.Len(t, servers, 1)
	assert.Equal(t, "everything", servers[0]["name"])
	assert.Greater(t, servers[0]["enabled_tool_count"], float64(0))

	showRes := callTool(t, c, "mcpjungle__show_server_tools", map[string]any{"server": "everything"})
	require.False(t, showRes.IsError, resultText(t, showRes))
	var shown []map[string]any
	require.NoError(t, json.Unmarshal([]byte(resultText(t, showRes)), &shown))
	assert.Contains(t, toolNames(shown), "everything__echo")

	// Activation adds the upstream tools to this session only.
	assert.Contains(t, listToolNames(t, c), "everything__echo")
	other := newLazyMCPClient(t, env, "/mcp", "")
	assert.NotContains(t, listToolNames(t, other), "everything__echo")

	direct := callTool(t, c, "everything__echo", map[string]any{"message": "lazy direct"})
	require.False(t, direct.IsError)
	assert.Contains(t, resultText(t, direct), "lazy direct")

	fallback := callTool(t, other, "mcpjungle__invoke", map[string]any{
		"name":      "everything__echo",
		"arguments": `{"message":"lazy fallback"}`,
	})
	require.False(t, fallback.IsError)
	assert.Contains(t, resultText(t, fallback), "lazy fallback")

	// The eager endpoint is unaffected by lazy-mode dispatch.
	eager := newMCPProxyClient(t, env, "")
	defer eager.Close()
	assert.Contains(t, listToolNames(t, eager), "everything__echo")
}

func TestE2E_LazyMode_EnterpriseAllowListEnforced(t *testing.T) {
	env := setupE2EServer(t, model.ModeEnterprise)
	registerEverythingServer(t, env, env.adminToken)

	deniedToken := createMcpClient(t, env, "lazy-denied", []string{"other"})
	denied := newLazyMCPClient(t, env, "/mcp", deniedToken)

	listRes := callTool(t, denied, "mcpjungle__list_servers", nil)
	require.False(t, listRes.IsError)
	assert.JSONEq(t, "[]", resultText(t, listRes))

	showRes := callTool(t, denied, "mcpjungle__show_server_tools", map[string]any{"server": "everything"})
	assert.True(t, showRes.IsError)
	assert.NotContains(t, listToolNames(t, denied), "everything__echo")

	invokeRes := callTool(t, denied, "mcpjungle__invoke", map[string]any{
		"name":      "everything__echo",
		"arguments": `{"message":"nope"}`,
	})
	assert.True(t, invokeRes.IsError)

	allowedToken := createMcpClient(t, env, "lazy-allowed", []string{"everything"})
	allowed := newLazyMCPClient(t, env, "/mcp", allowedToken)
	showRes = callTool(t, allowed, "mcpjungle__show_server_tools", map[string]any{"server": "everything"})
	require.False(t, showRes.IsError, resultText(t, showRes))
	echo := callTool(t, allowed, "everything__echo", map[string]any{"message": "allowed"})
	require.False(t, echo.IsError)
	assert.Contains(t, resultText(t, echo), "allowed")
}

func TestE2E_LazyMode_GroupEndpointLimitsTools(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodPost, "/api/v0/tool-groups", map[string]any{
		"name":           "lazy-group",
		"included_tools": []string{"everything__echo"},
	}, "")
	drain(resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	c := newLazyMCPClient(t, env, "/v0/groups/lazy-group/mcp", "")
	showRes := callTool(t, c, "mcpjungle__show_server_tools", map[string]any{"server": "everything"})
	require.False(t, showRes.IsError, resultText(t, showRes))
	var shown []map[string]any
	require.NoError(t, json.Unmarshal([]byte(resultText(t, showRes)), &shown))
	assert.Equal(t, []string{"everything__echo"}, toolNames(shown))
}
