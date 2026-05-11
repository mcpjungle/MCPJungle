package e2e_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/stretchr/testify/require"
)

func TestDashboardRootServedInDevMode(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)

	resp := env.do(t, http.MethodGet, "/", nil, "")
	defer drain(resp)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := readBody(t, resp)
	require.Contains(t, body, "MCPJungle Dashboard")
}

func TestDashboardRootHiddenInEnterpriseMode(t *testing.T) {
	env := setupE2EServer(t, model.ModeEnterprise)

	resp := env.do(t, http.MethodGet, "/", nil, env.adminToken)
	defer drain(resp)

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDashboardAPIHiddenInEnterpriseMode(t *testing.T) {
	env := setupE2EServer(t, model.ModeEnterprise)

	resp := env.do(t, http.MethodGet, "/api/dashboard/overview", nil, env.adminToken)
	defer drain(resp)

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDashboardAPIEmptyStates(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)

	overviewResp := env.do(t, http.MethodGet, "/api/dashboard/overview", nil, "")
	defer drain(overviewResp)
	require.Equal(t, http.StatusOK, overviewResp.StatusCode)

	var overview map[string]any
	decodeJSON(t, overviewResp, &overview)
	require.Equal(t, float64(0), overview["server_count"])
	require.NotNil(t, overview["empty_state"])

	serversResp := env.do(t, http.MethodGet, "/api/dashboard/servers", nil, "")
	defer drain(serversResp)
	require.Equal(t, http.StatusOK, serversResp.StatusCode)

	var servers map[string]any
	decodeJSON(t, serversResp, &servers)
	require.Empty(t, servers["servers"])
	require.NotNil(t, servers["empty_state"])
}

func TestDashboardAPIValidJSON(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	paths := []string{
		"/api/dashboard/overview",
		"/api/dashboard/servers",
		"/api/dashboard/tools",
		"/api/dashboard/prompts",
		"/api/dashboard/resources",
		"/api/dashboard/diagnostics",
	}

	for _, path := range paths {
		resp := env.do(t, http.MethodGet, path, nil, "")
		require.Equal(t, http.StatusOK, resp.StatusCode, path)
		var payload any
		decodeJSON(t, resp, &payload)
		drain(resp)
		require.NotNil(t, payload, path)
	}
}

func TestDashboardServerSummariesDoNotExposeSecrets(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)

	serverModel, err := model.NewStreamableHTTPServer(
		"secret-http",
		"contains a token",
		"https://example.com/mcp?api_key=top-secret",
		"bearer-token-value",
		map[string]string{
			"Authorization": "Bearer custom-secret",
			"X-Team":        "local-dev",
		},
		"",
	)
	require.NoError(t, err)
	require.NoError(t, env.db.Create(serverModel).Error)

	resp := env.do(t, http.MethodGet, "/api/dashboard/servers", nil, "")
	defer drain(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body := readBody(t, resp)
	require.NotContains(t, body, "bearer-token-value")
	require.NotContains(t, body, "custom-secret")
	require.NotContains(t, body, "api_key=top-secret")
	require.NotContains(t, strings.ToLower(body), "authorization")
	require.Contains(t, body, "\"header_keys\":[\"X-Team\"]")
}

func TestDashboardMutationsAndProxyExposure(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)

	registerResp := env.do(t, http.MethodPost, "/api/dashboard/servers", map[string]any{
		"name":        "dashsrv",
		"description": "Dashboard mutation test server",
		"transport":   "stdio",
		"command":     "npx",
		"args":        []string{"-y", "@modelcontextprotocol/server-everything", "stdio"},
	}, "")
	defer drain(registerResp)
	require.Equal(t, http.StatusCreated, registerResp.StatusCode)

	serversResp := env.do(t, http.MethodGet, "/api/dashboard/servers", nil, "")
	defer drain(serversResp)
	require.Equal(t, http.StatusOK, serversResp.StatusCode)
	var serversPayload map[string]any
	decodeJSON(t, serversResp, &serversPayload)
	servers := serversPayload["servers"].([]any)
	require.Len(t, servers, 1)
	server := servers[0].(map[string]any)
	require.Equal(t, "dashsrv", server["name"])
	require.Equal(t, true, server["enabled"])

	toolsResp := env.do(t, http.MethodGet, "/api/dashboard/tools", nil, "")
	defer drain(toolsResp)
	require.Equal(t, http.StatusOK, toolsResp.StatusCode)
	var toolsPayload map[string]any
	decodeJSON(t, toolsResp, &toolsPayload)
	require.NotEmpty(t, toolsPayload["tools"])
	firstTool := toolsPayload["tools"].([]any)[0].(map[string]any)
	require.Equal(t, true, firstTool["enabled"])

	promptsResp := env.do(t, http.MethodGet, "/api/dashboard/prompts", nil, "")
	defer drain(promptsResp)
	require.Equal(t, http.StatusOK, promptsResp.StatusCode)
	var promptsPayload map[string]any
	decodeJSON(t, promptsResp, &promptsPayload)
	require.NotEmpty(t, promptsPayload["prompts"])
	firstPrompt := promptsPayload["prompts"].([]any)[0].(map[string]any)
	require.Equal(t, true, firstPrompt["enabled"])

	proxyClient := newMCPProxyClient(t, env, "")
	toolsBefore, err := proxyClient.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	require.Contains(t, toolResultNames(toolsBefore.Tools), "dashsrv__echo")
	promptsBefore, err := proxyClient.ListPrompts(context.Background(), mcp.ListPromptsRequest{})
	require.NoError(t, err)
	require.Contains(t, promptResultNames(promptsBefore.Prompts), "dashsrv__simple-prompt")

	disableToolResp := env.do(t, http.MethodPatch, "/api/dashboard/tools/dashsrv__echo/enabled", map[string]any{
		"enabled": false,
	}, "")
	defer drain(disableToolResp)
	require.Equal(t, http.StatusOK, disableToolResp.StatusCode)
	toolsAfterDisable, err := proxyClient.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	require.NotContains(t, toolResultNames(toolsAfterDisable.Tools), "dashsrv__echo")

	disablePromptResp := env.do(t, http.MethodPatch, "/api/dashboard/prompts/dashsrv__simple-prompt/enabled", map[string]any{
		"enabled": false,
	}, "")
	defer drain(disablePromptResp)
	require.Equal(t, http.StatusOK, disablePromptResp.StatusCode)
	promptsAfterDisable, err := proxyClient.ListPrompts(context.Background(), mcp.ListPromptsRequest{})
	require.NoError(t, err)
	require.NotContains(t, promptResultNames(promptsAfterDisable.Prompts), "dashsrv__simple-prompt")

	disableServerResp := env.do(t, http.MethodPatch, "/api/dashboard/servers/dashsrv/enabled", map[string]any{
		"enabled": false,
	}, "")
	defer drain(disableServerResp)
	require.Equal(t, http.StatusOK, disableServerResp.StatusCode)
	toolsAfterServerDisable, err := proxyClient.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	require.NotContains(t, toolResultNames(toolsAfterServerDisable.Tools), "dashsrv__get-sum")
	promptsAfterServerDisable, err := proxyClient.ListPrompts(context.Background(), mcp.ListPromptsRequest{})
	require.NoError(t, err)
	require.NotContains(t, promptResultNames(promptsAfterServerDisable.Prompts), "dashsrv__simple-prompt")

	overviewResp := env.do(t, http.MethodGet, "/api/dashboard/overview", nil, "")
	defer drain(overviewResp)
	require.Equal(t, http.StatusOK, overviewResp.StatusCode)
	var overview map[string]any
	decodeJSON(t, overviewResp, &overview)
	require.Equal(t, float64(1), overview["server_count"])
	require.Equal(t, float64(0), overview["tool_count"])
	require.Equal(t, float64(0), overview["prompt_count"])

	enableServerResp := env.do(t, http.MethodPatch, "/api/dashboard/servers/dashsrv/enabled", map[string]any{
		"enabled": true,
	}, "")
	defer drain(enableServerResp)
	require.Equal(t, http.StatusOK, enableServerResp.StatusCode)

	enableToolResp := env.do(t, http.MethodPatch, "/api/dashboard/tools/dashsrv__echo/enabled", map[string]any{
		"enabled": true,
	}, "")
	defer drain(enableToolResp)
	require.Equal(t, http.StatusOK, enableToolResp.StatusCode)

	enablePromptResp := env.do(t, http.MethodPatch, "/api/dashboard/prompts/dashsrv__simple-prompt/enabled", map[string]any{
		"enabled": true,
	}, "")
	defer drain(enablePromptResp)
	require.Equal(t, http.StatusOK, enablePromptResp.StatusCode)

	toolsAfterEnable, err := proxyClient.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	require.Contains(t, toolResultNames(toolsAfterEnable.Tools), "dashsrv__echo")
	promptsAfterEnable, err := proxyClient.ListPrompts(context.Background(), mcp.ListPromptsRequest{})
	require.NoError(t, err)
	require.Contains(t, promptResultNames(promptsAfterEnable.Prompts), "dashsrv__simple-prompt")

	deleteResp := env.do(t, http.MethodDelete, "/api/dashboard/servers/dashsrv", nil, "")
	defer drain(deleteResp)
	require.Equal(t, http.StatusOK, deleteResp.StatusCode)

	finalServersResp := env.do(t, http.MethodGet, "/api/dashboard/servers", nil, "")
	defer drain(finalServersResp)
	require.Equal(t, http.StatusOK, finalServersResp.StatusCode)
	var finalServers map[string]any
	decodeJSON(t, finalServersResp, &finalServers)
	require.Empty(t, finalServers["servers"])
}

func toolResultNames(tools []mcp.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}

func promptResultNames(prompts []mcp.Prompt) []string {
	names := make([]string, 0, len(prompts))
	for _, prompt := range prompts {
		names = append(names, prompt.Name)
	}
	return names
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(body)
}
