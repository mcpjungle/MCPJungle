package e2e_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------
// Dev mode – global tools
// -----------------------------------------------------------------------

func TestE2E_DevMode_ListTools(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodGet, "/api/v0/tools", nil, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var tools []map[string]any
	decodeJSON(t, resp, &tools)

	names := toolNames(tools)
	assert.Contains(t, names, "everything__echo")
	assert.Contains(t, names, "everything__get-sum")
	assert.Contains(t, names, "everything__get-env")
}

func TestE2E_DevMode_GetTool(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodGet, "/api/v0/tool?name=everything__echo", nil, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var tool map[string]any
	decodeJSON(t, resp, &tool)
	assert.Equal(t, "everything__echo", tool["name"])
}

func TestE2E_DevMode_InvokeTool(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	body := map[string]any{
		"name":    "everything__echo",
		"message": "hello from e2e test",
	}
	resp := env.do(t, http.MethodPost, "/api/v0/tools/invoke", body, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result toolInvokeResult
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.Content, "echo response must have 'content' field")
	require.Equal(t, "text", result.Content[0].Type)
	require.Contains(t, result.Content[0].Text, "hello from e2e test", "echo tool must return the input message")
}

// -----------------------------------------------------------------------
// Dev mode – global prompts
// -----------------------------------------------------------------------

func TestE2E_DevMode_ListPrompts(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodGet, "/api/v0/prompts", nil, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var prompts []map[string]any
	decodeJSON(t, resp, &prompts)

	names := promptNames(prompts)
	assert.Contains(t, names, "everything__simple-prompt")
	assert.Contains(t, names, "everything__args-prompt")
}

func TestE2E_DevMode_GetPrompt(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	resp := env.do(t, http.MethodGet, "/api/v0/prompt?name=everything__simple-prompt", nil, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var prompt map[string]any
	decodeJSON(t, resp, &prompt)
	assert.Equal(t, "everything__simple-prompt", prompt["name"])
}

func TestE2E_DevMode_RenderPrompt(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	body := map[string]any{
		"name":      "everything__simple-prompt",
		"arguments": map[string]string{},
	}
	resp := env.do(t, http.MethodPost, "/api/v0/prompts/render", body, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result renderedPromptResult
	decodeJSON(t, resp, &result)
	require.NotEmpty(t, result.Messages, "rendered simple-prompt must have messages")
	require.Equal(t, "user", result.Messages[0].Role)
	require.Equal(t, "text", result.Messages[0].Content.Type)
	require.Equal(t, "This is a simple prompt without arguments.", result.Messages[0].Content.Text)
}

// -----------------------------------------------------------------------
// Dev mode – not-found error handling
// -----------------------------------------------------------------------

// TestE2E_DevMode_NotFound_DeregisterMissingServer verifies the full error propagation path:
// handler → service → ErrNotFound → handleServiceError → 404.
// Does not register any upstream server, so npx is not required.
func TestE2E_DevMode_NotFound_DeregisterMissingServer(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)

	resp := env.do(t, http.MethodDelete, "/api/v0/servers/nonexistent-server", nil, "")
	defer drain(resp)

	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	errMsg, ok := body["error"]
	require.True(t, ok, "response must contain 'error' field")
	require.Contains(t, string(errMsg), "not found")
}
