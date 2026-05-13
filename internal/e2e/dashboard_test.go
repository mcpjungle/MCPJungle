package e2e_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
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
		"/api/dashboard/tool-groups",
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

	toolsAfterServerDisableResp := env.do(t, http.MethodGet, "/api/dashboard/tools", nil, "")
	defer drain(toolsAfterServerDisableResp)
	require.Equal(t, http.StatusOK, toolsAfterServerDisableResp.StatusCode)
	var toolsAfterServerDisablePayload map[string]any
	decodeJSON(t, toolsAfterServerDisableResp, &toolsAfterServerDisablePayload)
	var echoTool map[string]any
	for _, item := range toolsAfterServerDisablePayload["tools"].([]any) {
		tool := item.(map[string]any)
		if tool["canonical_name"] == "dashsrv__echo" {
			echoTool = tool
			break
		}
	}
	require.NotNil(t, echoTool)
	require.Equal(t, false, echoTool["enabled"])
	require.Equal(t, false, echoTool["server_enabled"])

	promptsAfterServerDisableResp := env.do(t, http.MethodGet, "/api/dashboard/prompts", nil, "")
	defer drain(promptsAfterServerDisableResp)
	require.Equal(t, http.StatusOK, promptsAfterServerDisableResp.StatusCode)
	var promptsAfterServerDisablePayload map[string]any
	decodeJSON(t, promptsAfterServerDisableResp, &promptsAfterServerDisablePayload)
	var simplePrompt map[string]any
	for _, item := range promptsAfterServerDisablePayload["prompts"].([]any) {
		prompt := item.(map[string]any)
		if prompt["canonical_name"] == "dashsrv__simple-prompt" {
			simplePrompt = prompt
			break
		}
	}
	require.NotNil(t, simplePrompt)
	require.Equal(t, false, simplePrompt["enabled"])
	require.Equal(t, false, simplePrompt["server_enabled"])

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
	require.Greater(t, overview["tool_count"].(float64), float64(0))
	require.Greater(t, overview["prompt_count"].(float64), float64(0))

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

func TestDashboardToolGroupsCRUDAndValidation(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	listResp := env.do(t, http.MethodGet, "/api/dashboard/tool-groups", nil, "")
	defer drain(listResp)
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	var emptyPayload map[string]any
	decodeJSON(t, listResp, &emptyPayload)
	require.Empty(t, emptyPayload["tool_groups"])
	require.NotNil(t, emptyPayload["empty_state"])

	invalidResp := env.do(t, http.MethodPost, "/api/dashboard/tool-groups", map[string]any{
		"name":  "empty-group",
		"tools": []string{},
	}, "")
	defer drain(invalidResp)
	require.Equal(t, http.StatusBadRequest, invalidResp.StatusCode)

	createResp := env.do(t, http.MethodPost, "/api/dashboard/tool-groups", map[string]any{
		"name":        "coding",
		"description": "Coding helpers",
		"tools":       []string{"everything__echo", "everything__get-sum"},
	}, "")
	defer drain(createResp)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created map[string]any
	decodeJSON(t, createResp, &created)
	require.Equal(t, "coding", created["name"])
	require.Equal(t, float64(2), created["tool_count"])

	getResp := env.do(t, http.MethodGet, "/api/dashboard/tool-groups/coding", nil, "")
	defer drain(getResp)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var fetched map[string]any
	decodeJSON(t, getResp, &fetched)
	require.Equal(t, "coding", fetched["name"])
	tools := fetched["tools"].([]any)
	require.Len(t, tools, 2)

	deleteResp := env.do(t, http.MethodDelete, "/api/dashboard/tool-groups/coding", nil, "")
	defer drain(deleteResp)
	require.Equal(t, http.StatusOK, deleteResp.StatusCode)

	finalListResp := env.do(t, http.MethodGet, "/api/dashboard/tool-groups", nil, "")
	defer drain(finalListResp)
	require.Equal(t, http.StatusOK, finalListResp.StatusCode)
	var finalPayload map[string]any
	decodeJSON(t, finalListResp, &finalPayload)
	require.Empty(t, finalPayload["tool_groups"])
}

func TestDashboardRegisterServerHandlesOAuth(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	upstream := newMockOAuthMCPServer(t)

	registerResp := env.do(t, http.MethodPost, "/api/dashboard/servers", map[string]any{
		"name":        "oauthdash",
		"description": "Dashboard OAuth server",
		"transport":   "streamable_http",
		"url":         upstream.server.URL + "/mcp",
	}, "")
	defer drain(registerResp)
	require.Equal(t, http.StatusAccepted, registerResp.StatusCode)

	var registerPayload struct {
		AuthorizationRequired *types.UpstreamOAuthAuthorizationRequired `json:"authorization_required"`
	}
	decodeJSON(t, registerResp, &registerPayload)
	require.NotNil(t, registerPayload.AuthorizationRequired)

	sessionResp := env.do(
		t,
		http.MethodGet,
		"/api/dashboard/oauth/session/"+registerPayload.AuthorizationRequired.SessionID,
		nil,
		"",
	)
	defer drain(sessionResp)
	require.Equal(t, http.StatusOK, sessionResp.StatusCode)

	var pendingPayload map[string]any
	decodeJSON(t, sessionResp, &pendingPayload)
	require.Equal(t, "pending", pendingPayload["status"])

	authClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	authResp, err := authClient.Get(registerPayload.AuthorizationRequired.AuthorizationURL)
	require.NoError(t, err)
	defer authResp.Body.Close()
	require.Equal(t, http.StatusFound, authResp.StatusCode)

	callbackURL, err := url.Parse(authResp.Header.Get("Location"))
	require.NoError(t, err)

	callbackResp, err := http.Get(callbackURL.String())
	require.NoError(t, err)
	defer callbackResp.Body.Close()
	require.Equal(t, http.StatusOK, callbackResp.StatusCode)
	require.Contains(t, readBody(t, callbackResp), "Authorization successful")

	completedResp := env.do(
		t,
		http.MethodGet,
		"/api/dashboard/oauth/session/"+registerPayload.AuthorizationRequired.SessionID,
		nil,
		"",
	)
	defer drain(completedResp)
	require.Equal(t, http.StatusOK, completedResp.StatusCode)

	var completedPayload map[string]any
	decodeJSON(t, completedResp, &completedPayload)
	require.Equal(t, "completed", completedPayload["status"])

	serversResp := env.do(t, http.MethodGet, "/api/dashboard/servers", nil, "")
	defer drain(serversResp)
	require.Equal(t, http.StatusOK, serversResp.StatusCode)
	require.Contains(t, readBody(t, serversResp), "oauthdash")

	proxyClient := newMCPProxyClient(t, env, "")
	tools, err := proxyClient.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	require.Contains(t, toolResultNames(tools.Tools), "oauthdash__echo")
}

func TestDashboardOAuthCallbackMissingParams(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)

	resp := env.do(t, http.MethodGet, "/api/dashboard/oauth/callback", nil, "")
	defer drain(resp)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Contains(t, readBody(t, resp), "Missing required OAuth callback parameters")
}

func TestDashboardOAuthCallbackFailureIsTracked(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	upstream := newMockOAuthMCPServer(t)

	registerResp := env.do(t, http.MethodPost, "/api/dashboard/servers", map[string]any{
		"name":      "oauthfail",
		"transport": "streamable_http",
		"url":       upstream.server.URL + "/mcp",
	}, "")
	defer drain(registerResp)
	require.Equal(t, http.StatusAccepted, registerResp.StatusCode)

	var registerPayload struct {
		AuthorizationRequired *types.UpstreamOAuthAuthorizationRequired `json:"authorization_required"`
	}
	decodeJSON(t, registerResp, &registerPayload)
	require.NotNil(t, registerPayload.AuthorizationRequired)

	authURL, err := url.Parse(registerPayload.AuthorizationRequired.AuthorizationURL)
	require.NoError(t, err)
	state := authURL.Query().Get("state")
	require.NotEmpty(t, state)

	callbackResp := env.do(
		t,
		http.MethodGet,
		"/api/dashboard/oauth/callback?error=access_denied&state="+url.QueryEscape(state),
		nil,
		"",
	)
	defer drain(callbackResp)
	require.Equal(t, http.StatusBadRequest, callbackResp.StatusCode)
	require.Contains(t, readBody(t, callbackResp), "Authorization failed")

	sessionResp := env.do(
		t,
		http.MethodGet,
		"/api/dashboard/oauth/session/"+registerPayload.AuthorizationRequired.SessionID,
		nil,
		"",
	)
	defer drain(sessionResp)
	require.Equal(t, http.StatusOK, sessionResp.StatusCode)

	var sessionPayload map[string]any
	decodeJSON(t, sessionResp, &sessionPayload)
	require.Equal(t, "failed", sessionPayload["status"])
}

func TestDashboardOAuthSessionExpiresAndCleansUp(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	upstream := newMockOAuthMCPServer(t)

	registerResp := env.do(t, http.MethodPost, "/api/dashboard/servers", map[string]any{
		"name":      "oauthexpire",
		"transport": "streamable_http",
		"url":       upstream.server.URL + "/mcp",
	}, "")
	defer drain(registerResp)
	require.Equal(t, http.StatusAccepted, registerResp.StatusCode)

	var registerPayload struct {
		AuthorizationRequired *types.UpstreamOAuthAuthorizationRequired `json:"authorization_required"`
	}
	decodeJSON(t, registerResp, &registerPayload)
	require.NotNil(t, registerPayload.AuthorizationRequired)

	require.NoError(t, env.db.Model(&model.UpstreamOAuthPendingSession{}).
		Where("session_id = ?", registerPayload.AuthorizationRequired.SessionID).
		Update("expires_at", time.Now().Add(-time.Minute)).Error)

	sessionResp := env.do(
		t,
		http.MethodGet,
		"/api/dashboard/oauth/session/"+registerPayload.AuthorizationRequired.SessionID,
		nil,
		"",
	)
	defer drain(sessionResp)
	require.Equal(t, http.StatusOK, sessionResp.StatusCode)

	var sessionPayload map[string]any
	decodeJSON(t, sessionResp, &sessionPayload)
	require.Equal(t, "expired", sessionPayload["status"])

	var pendingCount int64
	require.NoError(t, env.db.Unscoped().Model(&model.UpstreamOAuthPendingSession{}).
		Where("session_id = ?", registerPayload.AuthorizationRequired.SessionID).
		Count(&pendingCount).Error)
	require.Zero(t, pendingCount)
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
