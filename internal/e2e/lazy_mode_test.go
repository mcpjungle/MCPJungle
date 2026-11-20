package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var lazyMetaTools = []string{
	"mcpjungle__list_tools",
	"mcpjungle__describe_tools",
	"mcpjungle__call_tool",
}

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

func resultEntries(t *testing.T, res *mcp.CallToolResult) []map[string]any {
	t.Helper()
	require.False(t, res.IsError, resultText(t, res))
	var entries []map[string]any
	require.NoError(t, json.Unmarshal([]byte(resultText(t, res)), &entries))
	return entries
}

func TestE2E_LazyMode_DiscoveryDescribeAndInvoke(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	c := newLazyMCPClient(t, env, "/mcp", "")

	assert.ElementsMatch(t, lazyMetaTools, listToolNames(t, c))

	listRes := callTool(t, c, "mcpjungle__list_tools", nil)
	entries := resultEntries(t, listRes)
	require.NotEmpty(t, entries, "the registered server must be discoverable")
	names := toolNames(entries)
	assert.Contains(t, names, "everything__echo")
	assert.NotContains(t, entries[0], "input_schema", "discovery must not expose schemas")

	describeRes := callTool(t, c, "mcpjungle__describe_tools", map[string]any{
		"names": []string{"everything__echo"},
	})
	described := resultEntries(t, describeRes)
	require.Len(t, described, 1)
	assert.Equal(t, "everything__echo", described[0]["name"])
	assert.NotEmpty(t, described[0]["input_schema"])

	callRes := callTool(t, c, "mcpjungle__call_tool", map[string]any{
		"name":      "everything__echo",
		"arguments": map[string]any{"message": "lazy call"},
	})
	require.False(t, callRes.IsError, resultText(t, callRes))
	assert.Contains(t, resultText(t, callRes), "lazy call")

	assert.ElementsMatch(t, lazyMetaTools, listToolNames(t, c))

	eager := newMCPProxyClient(t, env, "")
	defer eager.Close()
	assert.Contains(t, listToolNames(t, eager), "everything__echo")
}

func TestE2E_LazyMode_EnterpriseAllowListEnforced(t *testing.T) {
	env := setupE2EServer(t, model.ModeEnterprise)
	registerEverythingServer(t, env, env.adminToken)

	deniedToken := createMcpClient(t, env, "lazy-denied", []string{"other"})
	denied := newLazyMCPClient(t, env, "/mcp", deniedToken)

	listRes := callTool(t, denied, "mcpjungle__list_tools", nil)
	assert.JSONEq(t, "[]", resultText(t, listRes))

	describeRes := callTool(t, denied, "mcpjungle__describe_tools", map[string]any{
		"names": []string{"everything__echo"},
	})
	assert.True(t, describeRes.IsError)

	callRes := callTool(t, denied, "mcpjungle__call_tool", map[string]any{
		"name":      "everything__echo",
		"arguments": map[string]any{"message": "nope"},
	})
	assert.True(t, callRes.IsError)

	allowedToken := createMcpClient(t, env, "lazy-allowed", []string{"everything"})
	allowed := newLazyMCPClient(t, env, "/mcp", allowedToken)
	allowedList := callTool(t, allowed, "mcpjungle__list_tools", nil)
	assert.Contains(t, toolNames(resultEntries(t, allowedList)), "everything__echo")

	allowedCall := callTool(t, allowed, "mcpjungle__call_tool", map[string]any{
		"name":      "everything__echo",
		"arguments": map[string]any{"message": "allowed"},
	})
	require.False(t, allowedCall.IsError, resultText(t, allowedCall))
	assert.Contains(t, resultText(t, allowedCall), "allowed")
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

	listRes := callTool(t, c, "mcpjungle__list_tools", nil)
	assert.Equal(t, []string{"everything__echo"}, toolNames(resultEntries(t, listRes)))

	describeRes := callTool(t, c, "mcpjungle__describe_tools", map[string]any{
		"names": []string{"everything__echo"},
	})
	require.False(t, describeRes.IsError, resultText(t, describeRes))

	excluded := excludedEverythingTool(t, env)
	deniedCall := callTool(t, c, "mcpjungle__call_tool", map[string]any{
		"name":      excluded,
		"arguments": map[string]any{},
	})
	assert.True(t, deniedCall.IsError, "a tool outside the group must not be callable")

	deniedDescribe := callTool(t, c, "mcpjungle__describe_tools", map[string]any{
		"names": []string{excluded},
	})
	assert.True(t, deniedDescribe.IsError, "a tool outside the group must not be described")
}

func excludedEverythingTool(t *testing.T, env *e2eEnv) string {
	t.Helper()
	eager := newMCPProxyClient(t, env, "")
	defer eager.Close()

	for _, name := range listToolNames(t, eager) {
		if strings.HasPrefix(name, "everything__") && name != "everything__echo" {
			return name
		}
	}
	t.Skip("the everything server exposes no further tools")
	return ""
}
