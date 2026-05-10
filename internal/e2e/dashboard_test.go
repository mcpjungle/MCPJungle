package e2e_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

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

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(body)
}
