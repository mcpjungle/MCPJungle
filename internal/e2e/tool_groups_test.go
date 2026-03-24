package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------
// Dev mode – tool groups: CRUD
// -----------------------------------------------------------------------

func TestE2E_DevMode_ToolGroup_Create_Get_List_Delete(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	// Create
	resp := env.do(t, http.MethodPost, "/api/v0/tool-groups", map[string]any{
		"name":           "echogroup",
		"description":    "Only echo tool",
		"included_tools": []string{"everything__echo"},
	}, "")
	defer drain(resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp map[string]any
	decodeJSON(t, resp, &createResp)
	assert.Contains(t, createResp, "streamable_http_endpoint")

	// Get
	resp = env.do(t, http.MethodGet, "/api/v0/tool-groups/echogroup", nil, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var getResp map[string]any
	decodeJSON(t, resp, &getResp)
	assert.Equal(t, "echogroup", getResp["name"])

	// List
	resp = env.do(t, http.MethodGet, "/api/v0/tool-groups", nil, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var groups []map[string]any
	decodeJSON(t, resp, &groups)
	require.Len(t, groups, 1)
	assert.Equal(t, "echogroup", groups[0]["name"])

	// Delete
	resp = env.do(t, http.MethodDelete, "/api/v0/tool-groups/echogroup", nil, "")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	drain(resp)

	// Verify deletion
	resp = env.do(t, http.MethodGet, "/api/v0/tool-groups/echogroup", nil, "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	drain(resp)
}

// -----------------------------------------------------------------------
// Dev mode – tool groups: scoped operations via the group's MCP endpoint
// -----------------------------------------------------------------------

// TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_ListTools verifies that the tool
// group's own MCP endpoint returns only the tools explicitly included in the
// group, not all globally registered tools.
func TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_ListTools(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodPost, "/api/v0/tool-groups", map[string]any{
		"name":           "scoped-tools-group",
		"included_tools": []string{"everything__echo"},
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	drain(resp)

	c := newGroupMCPClient(t, env, "scoped-tools-group", "")
	result, err := c.ListTools(context.Background(), &mcp.ListToolsParams{})
	require.NoError(t, err)

	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	assert.Contains(t, names, "everything__echo", "group must expose the included tool")
	assert.NotContains(t, names, "everything__get-sum", "group must NOT expose tools outside its scope")
	assert.NotContains(t, names, "everything__get-env", "group must NOT expose tools outside its scope")
}

// TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_GetTool verifies that a specific
// tool in the group can be retrieved via the group's MCP endpoint and has the
// expected name and a non-empty description.
func TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_GetTool(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodPost, "/api/v0/tool-groups", map[string]any{
		"name":           "get-tool-group",
		"included_tools": []string{"everything__echo"},
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	drain(resp)

	c := newGroupMCPClient(t, env, "get-tool-group", "")
	result, err := c.ListTools(context.Background(), &mcp.ListToolsParams{})
	require.NoError(t, err)
	require.NotEmpty(t, result.Tools, "group must expose at least one tool")

	var echoTool *mcp.Tool
	for _, tool := range result.Tools {
		if tool.Name == "everything__echo" {
			echoTool = tool
			break
		}
	}
	require.NotNil(t, echoTool, "everything__echo must be present in the group")
	assert.Equal(t, "everything__echo", echoTool.Name)
	assert.NotEmpty(t, echoTool.Description, "echo tool must have a description")
}

// TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_InvokeTool verifies that a tool
// can be called through the group's MCP endpoint and returns the expected
// response content.
func TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_InvokeTool(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodPost, "/api/v0/tool-groups", map[string]any{
		"name":           "invoke-tool-group",
		"included_tools": []string{"everything__echo"},
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	drain(resp)

	c := newGroupMCPClient(t, env, "invoke-tool-group", "")
	result, err := c.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "everything__echo",
		Arguments: map[string]any{"message": "hello from group endpoint"},
	})
	require.NoError(t, err)
	require.False(t, result.IsError, "tool call must not return an error")
	require.NotEmpty(t, result.Content, "echo response must have content")

	first, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "first content item must be TextContent")
	assert.Contains(t, first.Text, "hello from group endpoint")
}

// TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_ListPrompts verifies prompt
// listing in the context of a tool group. Prompts are not group-scoped in the
// current implementation, so this test uses the global REST API and confirms
// that the expected server-everything prompts are still accessible.
func TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_ListPrompts(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodPost, "/api/v0/tool-groups", map[string]any{
		"name":           "prompts-group",
		"included_tools": []string{"everything__echo"},
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	drain(resp)

	resp = env.do(t, http.MethodGet, "/api/v0/prompts", nil, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var prompts []map[string]any
	decodeJSON(t, resp, &prompts)
	names := promptNames(prompts)
	assert.Contains(t, names, "everything__simple-prompt")
	assert.Contains(t, names, "everything__args-prompt")
}

// TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_GetPrompt verifies that a prompt
// can be retrieved by name via the global REST API in the context of a tool
// group setup.
func TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_GetPrompt(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodPost, "/api/v0/tool-groups", map[string]any{
		"name":           "get-prompt-group",
		"included_tools": []string{"everything__echo"},
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	drain(resp)

	resp = env.do(t, http.MethodGet, "/api/v0/prompt?name=everything__simple-prompt", nil, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var prompt map[string]any
	decodeJSON(t, resp, &prompt)
	assert.Equal(t, "everything__simple-prompt", prompt["name"])
}

// TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_RenderPrompt verifies that
// prompts can be rendered with full content assertions in the context of a
// tool group setup, mirroring the global render test.
func TestE2E_DevMode_ToolGroup_ViaGroupEndpoint_RenderPrompt(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodPost, "/api/v0/tool-groups", map[string]any{
		"name":           "render-prompt-group",
		"included_tools": []string{"everything__echo"},
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	drain(resp)

	resp = env.do(t, http.MethodPost, "/api/v0/prompts/render", map[string]any{
		"name":      "everything__simple-prompt",
		"arguments": map[string]string{},
	}, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var rendered renderedPromptResult
	decodeJSON(t, resp, &rendered)
	require.NotEmpty(t, rendered.Messages, "rendered simple-prompt must have messages")
	require.Equal(t, "user", rendered.Messages[0].Role)
	require.Equal(t, "text", rendered.Messages[0].Content.Type)
	require.Equal(t, "This is a simple prompt without arguments.", rendered.Messages[0].Content.Text)
}

// -----------------------------------------------------------------------
// Dev mode – tool groups: Streamable HTTP session persistence
// -----------------------------------------------------------------------

// TestE2E_DevMode_ToolGroup_StreamableHTTP_SessionPersistence is a regression
// test for the go-sdk migration bug where toolGroupMCPServerCallHandler was
// creating a new mcp.StreamableHTTPHandler on every HTTP request.
//
// In the official go-sdk, session state lives inside mcp.StreamableHTTPHandler
// (not in mcp.Server). Re-creating the handler per request meant that the
// session map was always empty when the second request arrived, causing any
// call after initialize (e.g. tools/call) to fail with "unknown session".
//
// In the original mark3labs SDK this was not a problem because session state
// was stored inside the MCPServer struct itself; the per-request HTTP wrapper
// was stateless, so discarding it was safe.
//
// Reproduction pattern:
//  1. Client A POSTs initialize  → fresh handler H1 is created and cached,
//     session S_A is stored in H1.
//  2. Client B POSTs initialize  → H1 is reused (fix) or H2 is created (bug),
//     session S_B is stored in H1/H2.
//  3. Client A POSTs tools/call  → with the bug H3 is created (no S_A) → error;
//     with the fix H1 is reused (S_A present) → success.
//  4. Same for Client B.
//
// The test connects two independent MCP clients and verifies that both can
// invoke a tool after initialization, proving session state is preserved.
func TestE2E_DevMode_ToolGroup_StreamableHTTP_SessionPersistence(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodPost, "/api/v0/tool-groups", map[string]any{
		"name":           "session-test-group",
		"included_tools": []string{"everything__echo"},
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	drain(resp)

	// Connect two independent clients.  Each Connect() call performs the MCP
	// initialize handshake (one HTTP round-trip) and stores the returned session
	// ID in the client's transport.  The subsequent CallTool sends a second
	// request carrying that session ID — which must be recognised by the same
	// handler instance.
	c1 := newGroupMCPClient(t, env, "session-test-group", "")
	c2 := newGroupMCPClient(t, env, "session-test-group", "")

	result1, err := c1.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "everything__echo",
		Arguments: map[string]any{"message": "session-one"},
	})
	require.NoError(t, err, "client 1: tool call must succeed — handler must be reused so session S1 is still known")
	require.False(t, result1.IsError)

	result2, err := c2.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "everything__echo",
		Arguments: map[string]any{"message": "session-two"},
	})
	require.NoError(t, err, "client 2: tool call must succeed — handler must be reused so session S2 is still known")
	require.False(t, result2.IsError)

	first1, ok := result1.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, first1.Text, "session-one", "response must echo client 1's message")

	first2, ok := result2.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, first2.Text, "session-two", "response must echo client 2's message")
}
