// Package live_test runs integration tests against an already-running MCPJungle instance.
//
// Use scripts/run-live-tests.sh to start all required services and run this suite,
// or set environment variables manually and run:
//
//	go test ./internal/e2e/live/ -run TestLive -v -timeout 120s
//
// Required env vars (set by scripts/run-live-tests.sh):
//
//	MCPJUNGLE_DEV_BASE        MCPJungle dev instance URL  (default: http://localhost:8080)
//	MCPJUNGLE_ENT_BASE        MCPJungle enterprise URL    (default: http://localhost:8081)
//	MCPJUNGLE_ENT_ADMIN_TOKEN Admin token for enterprise instance
//	MCPJUNGLE_ENT_CLIENT_TOKEN Client token with allow_list=["everything"]
//	MCPJUNGLE_ENT_BLOCKED_TOKEN Client token with allow_list=["other-server"] (no real access)
//
// If MCPJUNGLE_ENT_ADMIN_TOKEN is not set, enterprise tests are skipped.
// If MCPJUNGLE_DEV_BASE is unreachable, all dev tests are skipped.
package live_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------
// Env-var config
// -----------------------------------------------------------------------

func liveDevBase() string {
	if v := os.Getenv("MCPJUNGLE_DEV_BASE"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func liveEntBase() string {
	if v := os.Getenv("MCPJUNGLE_ENT_BASE"); v != "" {
		return v
	}
	return "http://localhost:8081"
}

func liveEntAdminToken() string   { return os.Getenv("MCPJUNGLE_ENT_ADMIN_TOKEN") }
func liveEntClientToken() string  { return os.Getenv("MCPJUNGLE_ENT_CLIENT_TOKEN") }
func liveEntBlockedToken() string { return os.Getenv("MCPJUNGLE_ENT_BLOCKED_TOKEN") }

// -----------------------------------------------------------------------
// Live env (wraps a running MCPJungle instance)
// -----------------------------------------------------------------------

type liveEnv struct{ base string }

func (e *liveEnv) do(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()
	return e.doWithToken(t, method, path, body, "")
}

func (e *liveEnv) doWithToken(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()
	var reqBody *strings.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = strings.NewReader(string(b))
	}
	var req *http.Request
	var err error
	if reqBody != nil {
		req, err = http.NewRequest(method, e.base+path, reqBody)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, e.base+path, nil)
		require.NoError(t, err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func skipIfDown(t *testing.T) *liveEnv {
	t.Helper()
	base := liveDevBase()
	resp, err := http.Get(base + "/health")
	if err != nil || resp.StatusCode != 200 {
		t.Skipf("MCPJungle dev not reachable at %s – skipping live tests", base)
	}
	resp.Body.Close()
	return &liveEnv{base: base}
}

func skipIfEnterpriseDown(t *testing.T) *liveEnv {
	t.Helper()
	if liveEntAdminToken() == "" {
		t.Skip("MCPJUNGLE_ENT_ADMIN_TOKEN not set – skipping enterprise live tests")
	}
	base := liveEntBase()
	resp, err := http.Get(base + "/health")
	if err != nil || resp.StatusCode != 200 {
		t.Skipf("MCPJungle enterprise not reachable at %s – skipping", base)
	}
	resp.Body.Close()
	return &liveEnv{base: base}
}

// -----------------------------------------------------------------------
// Section 1 – REST API
// -----------------------------------------------------------------------

func TestLive_REST_ListServers(t *testing.T) {
	env := skipIfDown(t)
	resp := env.do(t, http.MethodGet, "/api/v0/servers", nil)
	require.Equal(t, 200, resp.StatusCode)
	var servers []map[string]any
	decodeJSON(t, resp, &servers)
	names := make([]string, 0, len(servers))
	for _, s := range servers {
		names = append(names, s["name"].(string))
	}
	t.Logf("registered servers: %v", names)
	assert.Contains(t, names, "everything", "stdio server must be registered")
	assert.Contains(t, names, "everything-sse", "SSE server must be registered")
	assert.Contains(t, names, "everything-http", "streamable_http server must be registered")
}

func TestLive_REST_ListTools(t *testing.T) {
	env := skipIfDown(t)
	resp := env.do(t, http.MethodGet, "/api/v0/tools", nil)
	require.Equal(t, 200, resp.StatusCode)
	var tools []map[string]any
	decodeJSON(t, resp, &tools)
	t.Logf("total tools: %d", len(tools))
	names := toolNames(tools)
	assert.Contains(t, names, "everything__echo")
	assert.Contains(t, names, "everything-sse__echo")
	assert.Contains(t, names, "everything-http__echo")
}

func TestLive_REST_GetTool(t *testing.T) {
	env := skipIfDown(t)
	resp := env.do(t, http.MethodGet, "/api/v0/tool?name=everything__echo", nil)
	require.Equal(t, 200, resp.StatusCode)
	var tool map[string]any
	decodeJSON(t, resp, &tool)
	assert.Equal(t, "everything__echo", tool["name"])
	assert.NotEmpty(t, tool["description"])
}

func TestLive_REST_InvokeTool_Stdio(t *testing.T) {
	env := skipIfDown(t)
	resp := env.do(t, http.MethodPost, "/api/v0/tools/invoke", map[string]any{
		"name": "everything__get-sum", "a": 42, "b": 58,
	})
	require.Equal(t, 200, resp.StatusCode)
	var result toolInvokeResult
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.Content)
	t.Logf("stdio get-sum: %s", result.Content[0].Text)
	assert.Contains(t, result.Content[0].Text, "100")
}

func TestLive_REST_InvokeTool_SSE(t *testing.T) {
	env := skipIfDown(t)
	resp := env.do(t, http.MethodPost, "/api/v0/tools/invoke", map[string]any{
		"name": "everything-sse__get-sum", "a": 10, "b": 90,
	})
	require.Equal(t, 200, resp.StatusCode)
	var result toolInvokeResult
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.Content)
	t.Logf("sse get-sum: %s", result.Content[0].Text)
	assert.Contains(t, result.Content[0].Text, "100")
}

func TestLive_REST_InvokeTool_StreamableHTTP(t *testing.T) {
	env := skipIfDown(t)
	resp := env.do(t, http.MethodPost, "/api/v0/tools/invoke", map[string]any{
		"name": "everything-http__get-sum", "a": 33, "b": 67,
	})
	require.Equal(t, 200, resp.StatusCode)
	var result toolInvokeResult
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.Content)
	t.Logf("streamable_http get-sum: %s", result.Content[0].Text)
	assert.Contains(t, result.Content[0].Text, "100")
}

func TestLive_REST_ListPrompts(t *testing.T) {
	env := skipIfDown(t)
	resp := env.do(t, http.MethodGet, "/api/v0/prompts", nil)
	require.Equal(t, 200, resp.StatusCode)
	var prompts []map[string]any
	decodeJSON(t, resp, &prompts)
	t.Logf("total prompts: %d", len(prompts))
	names := promptNames(prompts)
	assert.Contains(t, names, "everything__simple-prompt")
	assert.Contains(t, names, "everything-sse__simple-prompt")
	assert.Contains(t, names, "everything-http__simple-prompt")
}

func TestLive_REST_RenderPrompt(t *testing.T) {
	env := skipIfDown(t)
	resp := env.do(t, http.MethodPost, "/api/v0/prompts/render", map[string]any{
		"name": "everything__simple-prompt", "arguments": map[string]string{},
	})
	require.Equal(t, 200, resp.StatusCode)
	var result renderedPromptResult
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.Messages)
	t.Logf("rendered prompt: %s", result.Messages[0].Content.Text)
	assert.Equal(t, "This is a simple prompt without arguments.", result.Messages[0].Content.Text)
}

// -----------------------------------------------------------------------
// Section 2 – /mcp (Streamable HTTP proxy)
// -----------------------------------------------------------------------

func TestLive_MCP_ListTools_StdioAndHTTP(t *testing.T) {
	skipIfDown(t)
	cs := liveMCPClient(t, liveDevBase()+"/mcp")
	result, err := cs.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	names := make([]string, 0, len(result.Tools))
	for _, tl := range result.Tools {
		names = append(names, tl.Name)
	}
	t.Logf("/mcp tool count: %d", len(result.Tools))
	assert.Contains(t, names, "everything__echo", "stdio tool must be on /mcp")
	assert.Contains(t, names, "everything-http__echo", "streamable_http tool must be on /mcp")
	assert.NotContains(t, names, "everything-sse__echo", "SSE tool must NOT be on /mcp")
}

func TestLive_MCP_CallTool_Stdio(t *testing.T) {
	skipIfDown(t)
	cs := liveMCPClient(t, liveDevBase()+"/mcp")
	result, err := cs.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "everything__get-sum", Arguments: map[string]any{"a": 5, "b": 5}},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
	tc, ok := mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	t.Logf("stdio CallTool: %s", tc.Text)
	assert.Contains(t, tc.Text, "10")
}

func TestLive_MCP_CallTool_StreamableHTTP(t *testing.T) {
	skipIfDown(t)
	cs := liveMCPClient(t, liveDevBase()+"/mcp")
	result, err := cs.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "everything-http__get-sum", Arguments: map[string]any{"a": 20, "b": 80}},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
	tc, ok := mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	t.Logf("streamable_http CallTool: %s", tc.Text)
	assert.Contains(t, tc.Text, "100")
}

func TestLive_MCP_CallTool_Echo(t *testing.T) {
	skipIfDown(t)
	cs := liveMCPClient(t, liveDevBase()+"/mcp")
	result, err := cs.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "everything__echo", Arguments: map[string]any{"message": "migration works"}},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
	tc, ok := mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	t.Logf("echo: %s", tc.Text)
	assert.Contains(t, tc.Text, "migration works")
}

func TestLive_MCP_ListPrompts(t *testing.T) {
	skipIfDown(t)
	cs := liveMCPClient(t, liveDevBase()+"/mcp")
	result, err := cs.ListPrompts(context.Background(), mcp.ListPromptsRequest{})
	require.NoError(t, err)
	names := make([]string, 0, len(result.Prompts))
	for _, p := range result.Prompts {
		names = append(names, p.Name)
	}
	t.Logf("/mcp prompt count: %d", len(result.Prompts))
	assert.Contains(t, names, "everything__simple-prompt")
	assert.Contains(t, names, "everything-http__simple-prompt")
}

func TestLive_MCP_GetPrompt(t *testing.T) {
	skipIfDown(t)
	cs := liveMCPClient(t, liveDevBase()+"/mcp")
	result, err := cs.GetPrompt(context.Background(), mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{Name: "everything__simple-prompt"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Messages)
	tc, ok := mcp.AsTextContent(result.Messages[0].Content)
	require.True(t, ok)
	t.Logf("prompt: %s", tc.Text)
	assert.Equal(t, "This is a simple prompt without arguments.", tc.Text)
}

// -----------------------------------------------------------------------
// Section 3 – /sse (legacy SSE proxy)
// -----------------------------------------------------------------------

func TestLive_SSE_ListTools(t *testing.T) {
	skipIfDown(t)
	cs := liveMCPClient(t, liveDevBase()+"/sse")
	result, err := cs.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	names := make([]string, 0, len(result.Tools))
	for _, tl := range result.Tools {
		names = append(names, tl.Name)
	}
	t.Logf("/sse tool count: %d", len(result.Tools))
	assert.Contains(t, names, "everything-sse__echo", "SSE-upstream tool must be on /sse")
	assert.NotContains(t, names, "everything__echo", "stdio tool must NOT be on /sse")
	assert.NotContains(t, names, "everything-http__echo", "streamable_http tool must NOT be on /sse")
}

func TestLive_SSE_CallTool(t *testing.T) {
	skipIfDown(t)
	cs := liveMCPClient(t, liveDevBase()+"/sse")
	result, err := cs.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "everything-sse__get-sum", Arguments: map[string]any{"a": 7, "b": 3}},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)
	tc, ok := mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	t.Logf("SSE CallTool: %s", tc.Text)
	assert.Contains(t, tc.Text, "10")
}

// -----------------------------------------------------------------------
// Section 4 – Tool Groups
// -----------------------------------------------------------------------

func TestLive_ToolGroup_CRUD_And_MCP(t *testing.T) {
	env := skipIfDown(t)
	groupName := "live-test-group"

	env.do(t, http.MethodDelete, "/api/v0/tool-groups/"+groupName, nil)
	t.Cleanup(func() { env.do(t, http.MethodDelete, "/api/v0/tool-groups/"+groupName, nil) })

	resp := env.do(t, http.MethodPost, "/api/v0/tool-groups", map[string]any{
		"name":           groupName,
		"description":    "live test group",
		"included_tools": []string{"everything__get-sum", "everything-http__echo"},
	})
	require.Equal(t, 201, resp.StatusCode)
	var created map[string]any
	decodeJSON(t, resp, &created)
	t.Logf("group endpoint: %v", created["streamable_http_endpoint"])
	assert.NotEmpty(t, created["streamable_http_endpoint"])

	resp = env.do(t, http.MethodGet, "/api/v0/tool-groups/"+groupName, nil)
	require.Equal(t, 200, resp.StatusCode)
	var got map[string]any
	decodeJSON(t, resp, &got)
	assert.Equal(t, groupName, got["name"])

	cs := liveMCPClient(t, fmt.Sprintf("%s/v0/groups/%s/mcp", liveDevBase(), groupName))

	listResult, err := cs.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	names := make([]string, 0, len(listResult.Tools))
	for _, tl := range listResult.Tools {
		names = append(names, tl.Name)
	}
	t.Logf("group tools: %v", names)
	assert.Contains(t, names, "everything__get-sum")
	assert.Contains(t, names, "everything-http__echo")
	assert.NotContains(t, names, "everything__echo", "non-included tool must be absent")

	callResult, err := cs.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "everything__get-sum", Arguments: map[string]any{"a": 25, "b": 75}},
	})
	require.NoError(t, err)
	require.False(t, callResult.IsError)
	tc, ok := mcp.AsTextContent(callResult.Content[0])
	require.True(t, ok)
	t.Logf("group CallTool: %s", tc.Text)
	assert.Contains(t, tc.Text, "100")
}

func TestLive_ToolGroup_SSE_Endpoint(t *testing.T) {
	env := skipIfDown(t)
	groupName := "live-sse-group"

	env.do(t, http.MethodDelete, "/api/v0/tool-groups/"+groupName, nil)
	t.Cleanup(func() { env.do(t, http.MethodDelete, "/api/v0/tool-groups/"+groupName, nil) })

	resp := env.do(t, http.MethodPost, "/api/v0/tool-groups", map[string]any{
		"name":           groupName,
		"included_tools": []string{"everything-sse__get-sum"},
	})
	require.Equal(t, 201, resp.StatusCode)
	drain(resp)

	cs := liveMCPClient(t, fmt.Sprintf("%s/v0/groups/%s/sse", liveDevBase(), groupName))

	listResult, err := cs.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	names := make([]string, 0, len(listResult.Tools))
	for _, tl := range listResult.Tools {
		names = append(names, tl.Name)
	}
	t.Logf("group SSE tools: %v", names)
	assert.Contains(t, names, "everything-sse__get-sum")

	callResult, err := cs.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "everything-sse__get-sum", Arguments: map[string]any{"a": 11, "b": 89}},
	})
	require.NoError(t, err)
	require.False(t, callResult.IsError)
	tc, ok := mcp.AsTextContent(callResult.Content[0])
	require.True(t, ok)
	t.Logf("group SSE CallTool: %s", tc.Text)
	assert.Contains(t, tc.Text, "100")
}

// -----------------------------------------------------------------------
// Section 5 – Enterprise Mode
// -----------------------------------------------------------------------

func TestLive_Enterprise_Unauthenticated_Returns401(t *testing.T) {
	env := skipIfEnterpriseDown(t)
	for _, path := range []string{"/api/v0/tools", "/api/v0/prompts", "/api/v0/servers"} {
		r := env.do(t, http.MethodGet, path, nil)
		drain(r)
		assert.Equal(t, 401, r.StatusCode, "unauthenticated %s must be 401", path)
	}
}

func TestLive_Enterprise_MCP_RequiresClientToken(t *testing.T) {
	env := skipIfEnterpriseDown(t)
	for _, tok := range []string{"", liveEntAdminToken()} {
		r := env.doWithToken(t, http.MethodGet, "/mcp", nil, tok)
		drain(r)
		assert.Equal(t, 401, r.StatusCode, "token %q must not grant /mcp access", tok)
	}
	clientToken := liveEntClientToken()
	if clientToken == "" {
		t.Skip("MCPJUNGLE_ENT_CLIENT_TOKEN not set")
	}
	r := env.doWithToken(t, http.MethodPost, "/mcp", nil, clientToken)
	drain(r)
	assert.NotEqual(t, 401, r.StatusCode, "valid client token must not return 401")
}

func TestLive_Enterprise_ACL_AllowedClient_SeesTools(t *testing.T) {
	skipIfEnterpriseDown(t)
	clientToken := liveEntClientToken()
	if clientToken == "" {
		t.Skip("MCPJUNGLE_ENT_CLIENT_TOKEN not set")
	}
	cs := liveMCPClientWithToken(t, liveEntBase()+"/mcp", clientToken)
	result, err := cs.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	names := make([]string, 0, len(result.Tools))
	for _, tl := range result.Tools {
		names = append(names, tl.Name)
	}
	t.Logf("allowed client sees %d tools: %v", len(result.Tools), names)
	assert.Contains(t, names, "everything__echo", "allowed server tools must be visible")
}

func TestLive_Enterprise_ACL_BlockedClient_SeesNoTools(t *testing.T) {
	skipIfEnterpriseDown(t)
	blockedToken := liveEntBlockedToken()
	if blockedToken == "" {
		t.Skip("MCPJUNGLE_ENT_BLOCKED_TOKEN not set")
	}
	cs := liveMCPClientWithToken(t, liveEntBase()+"/mcp", blockedToken)
	result, err := cs.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	t.Logf("blocked client sees %d tools", len(result.Tools))
	assert.Empty(t, result.Tools, "blocked client must see no tools")
}

func TestLive_Enterprise_ACL_BlockedClient_CallToolReturnsError(t *testing.T) {
	skipIfEnterpriseDown(t)
	blockedToken := liveEntBlockedToken()
	if blockedToken == "" {
		t.Skip("MCPJUNGLE_ENT_BLOCKED_TOKEN not set")
	}
	cs := liveMCPClientWithToken(t, liveEntBase()+"/mcp", blockedToken)
	_, err := cs.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "everything__echo", Arguments: map[string]any{"message": "should be blocked"}},
	})
	assert.Error(t, err, "blocked client calling restricted tool must return error")
	t.Logf("blocked call error (expected): %v", err)
}
