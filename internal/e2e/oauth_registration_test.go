package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOAuthMCPServer struct {
	server            *httptest.Server
	accessToken       string
	registerCalls     int
	mcpAuthHeaders    []string
	authorizedMCPHits int
}

func newMockOAuthMCPServer(t *testing.T) *mockOAuthMCPServer {
	t.Helper()

	upstreamMCP := mcpserver.NewMCPServer("oauth-upstream", "0.1.0")
	upstreamMCP.AddTool(
		mcp.NewTool("echo", mcp.WithDescription("Echoes the msg argument"), mcp.WithString("msg", mcp.Required())),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, _ := req.GetArguments()["msg"].(string)
			return mcp.NewToolResultText("oauth echo: " + msg), nil
		},
	)
	streamable := mcpserver.NewStreamableHTTPServer(upstreamMCP)

	mock := &mockOAuthMCPServer{accessToken: "mock-access-token"}
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/oauth-protected-resource/mcp", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"authorization_servers": []string{mock.server.URL},
			"resource":              mock.server.URL + "/mcp",
		})
	})

	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 mock.server.URL,
			"authorization_endpoint": mock.server.URL + "/authorize",
			"token_endpoint":         mock.server.URL + "/token",
			"registration_endpoint":  mock.server.URL + "/register",
			"response_types_supported": []string{
				"code",
			},
		})
	})

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		mock.registerCalls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"client_id": "mock-client-id",
		})
	})

	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")
		u, err := url.Parse(redirectURI)
		require.NoError(t, err)
		q := u.Query()
		q.Set("code", "mock-auth-code")
		q.Set("state", state)
		u.RawQuery = q.Encode()
		http.Redirect(w, r, u.String(), http.StatusFound)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.Equal(t, "mock-auth-code", r.Form.Get("code"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  mock.accessToken,
			"token_type":    "Bearer",
			"refresh_token": "mock-refresh-token",
			"expires_in":    3600,
			"scope":         "mcp.read",
		})
	})

	mux.Handle("/mcp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mcpAuthHeaders = append(mock.mcpAuthHeaders, r.Header.Get("Authorization"))
		if r.Header.Get("Authorization") != "Bearer "+mock.accessToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}
		mock.authorizedMCPHits++
		streamable.ServeHTTP(w, r)
	}))

	mock.server = httptest.NewServer(mux)
	t.Cleanup(mock.server.Close)
	return mock
}

type mockPublicMCPServer struct {
	server           *httptest.Server
	authHeaders      []string
	requestsWithAuth int
}

func newMockPublicMCPServer(t *testing.T) *mockPublicMCPServer {
	t.Helper()

	upstreamMCP := mcpserver.NewMCPServer("public-upstream", "0.1.0")
	upstreamMCP.AddTool(
		mcp.NewTool("echo", mcp.WithDescription("Echoes the msg argument"), mcp.WithString("msg", mcp.Required())),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, _ := req.GetArguments()["msg"].(string)
			return mcp.NewToolResultText("public echo: " + msg), nil
		},
	)
	streamable := mcpserver.NewStreamableHTTPServer(upstreamMCP)

	mock := &mockPublicMCPServer{}
	mux := http.NewServeMux()
	mux.Handle("/mcp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		mock.authHeaders = append(mock.authHeaders, authHeader)
		if authHeader != "" {
			mock.requestsWithAuth++
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unexpected authorization header"))
			return
		}
		streamable.ServeHTTP(w, r)
	}))

	mock.server = httptest.NewServer(mux)
	t.Cleanup(mock.server.Close)
	return mock
}

func startOAuthRegistration(t *testing.T, env *e2eEnv, upstreamURL string, force bool) types.RegisterServerResult {
	t.Helper()

	path := "/api/v0/servers"
	if force {
		path += "?force=true"
	}
	resp := env.do(t, http.MethodPost, path, map[string]any{
		"name":               "oauthsrv",
		"description":        "OAuth-protected upstream MCP server",
		"transport":          "streamable_http",
		"url":                upstreamURL,
		"oauth_redirect_uri": "http://127.0.0.1:9999/oauth/callback",
		"oauth_scopes":       []string{"mcp.read"},
	}, "")
	defer drain(resp)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	var registerResult types.RegisterServerResult
	decodeJSON(t, resp, &registerResult)
	require.NotNil(t, registerResult.AuthorizationRequired)
	return registerResult
}

func authorizeAndCompleteOAuth(t *testing.T, env *e2eEnv, registerResult types.RegisterServerResult, stateOverride string) *http.Response {
	t.Helper()

	authClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	authResp, err := authClient.Get(registerResult.AuthorizationRequired.AuthorizationURL)
	require.NoError(t, err)
	defer authResp.Body.Close()
	require.Equal(t, http.StatusFound, authResp.StatusCode)

	callbackURL, err := url.Parse(authResp.Header.Get("Location"))
	require.NoError(t, err)

	state := callbackURL.Query().Get("state")
	if stateOverride != "" {
		state = stateOverride
	}

	return env.do(
		t,
		http.MethodPost,
		"/api/v0/upstream_oauth/sessions/"+registerResult.AuthorizationRequired.SessionID+"/complete",
		map[string]any{
			"code":  callbackURL.Query().Get("code"),
			"state": state,
		},
		"",
	)
}

func TestE2E_DevMode_RegisterOAuthHTTPServerAndInvokeTool(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	upstream := newMockOAuthMCPServer(t)

	registerResult := startOAuthRegistration(t, env, upstream.server.URL+"/mcp", false)
	completeResp := authorizeAndCompleteOAuth(t, env, registerResult, "")
	defer drain(completeResp)
	require.Equal(t, http.StatusCreated, completeResp.StatusCode)

	var completeResult types.RegisterServerResult
	decodeJSON(t, completeResp, &completeResult)
	require.NotNil(t, completeResult.Server)
	assert.Equal(t, "oauthsrv", completeResult.Server.Name)

	listToolsResp := env.do(t, http.MethodGet, "/api/v0/tools", nil, "")
	defer drain(listToolsResp)
	require.Equal(t, http.StatusOK, listToolsResp.StatusCode)
	var tools []map[string]any
	decodeJSON(t, listToolsResp, &tools)
	assert.Contains(t, toolNames(tools), "oauthsrv__echo")

	for _, msg := range []string{"hello oauth", "hello again"} {
		invokeResp := env.do(t, http.MethodPost, "/api/v0/tools/invoke", map[string]any{
			"name": "oauthsrv__echo",
			"msg":  msg,
		}, "")
		defer drain(invokeResp)
		require.Equal(t, http.StatusOK, invokeResp.StatusCode)

		var invokeResult toolInvokeResult
		decodeJSON(t, invokeResp, &invokeResult)
		require.NotEmpty(t, invokeResult.Content)
		assert.Contains(t, invokeResult.Content[0].Text, msg)
	}

	assert.GreaterOrEqual(t, upstream.authorizedMCPHits, 2)
}

func TestE2E_DevMode_CompleteOAuthSessionRejectsWrongState(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	upstream := newMockOAuthMCPServer(t)

	registerResult := startOAuthRegistration(t, env, upstream.server.URL+"/mcp", false)
	completeResp := authorizeAndCompleteOAuth(t, env, registerResult, "wrong-state")
	defer drain(completeResp)
	require.Equal(t, http.StatusBadRequest, completeResp.StatusCode)
}

func TestE2E_DevMode_CompleteOAuthSessionRejectsExpiredPendingSession(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	upstream := newMockOAuthMCPServer(t)

	registerResult := startOAuthRegistration(t, env, upstream.server.URL+"/mcp", false)
	require.NoError(t, env.db.Model(&model.UpstreamOAuthPendingSession{}).
		Where("session_id = ?", registerResult.AuthorizationRequired.SessionID).
		Update("expires_at", time.Now().Add(-time.Minute)).Error)

	completeResp := authorizeAndCompleteOAuth(t, env, registerResult, "")
	defer drain(completeResp)
	require.Equal(t, http.StatusBadRequest, completeResp.StatusCode)
}

func TestE2E_DevMode_ForceReRegisterOAuthServer(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	upstream := newMockOAuthMCPServer(t)

	first := startOAuthRegistration(t, env, upstream.server.URL+"/mcp", false)
	firstComplete := authorizeAndCompleteOAuth(t, env, first, "")
	defer drain(firstComplete)
	require.Equal(t, http.StatusCreated, firstComplete.StatusCode)

	second := startOAuthRegistration(t, env, upstream.server.URL+"/mcp", true)
	secondComplete := authorizeAndCompleteOAuth(t, env, second, "")
	defer drain(secondComplete)
	require.Equal(t, http.StatusCreated, secondComplete.StatusCode)

	listServersResp := env.do(t, http.MethodGet, "/api/v0/servers", nil, "")
	defer drain(listServersResp)
	require.Equal(t, http.StatusOK, listServersResp.StatusCode)

	var servers []*types.McpServer
	decodeJSON(t, listServersResp, &servers)

	oauthCount := 0
	for _, server := range servers {
		if server.Name == "oauthsrv" {
			oauthCount++
		}
	}
	require.Equal(t, 1, oauthCount)
}

func TestE2E_DevMode_PublicHTTPServerAfterOAuthDoesNotReceiveAuthorizationHeader(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	oauthUpstream := newMockOAuthMCPServer(t)
	publicUpstream := newMockPublicMCPServer(t)

	registerResult := startOAuthRegistration(t, env, oauthUpstream.server.URL+"/mcp", false)
	completeResp := authorizeAndCompleteOAuth(t, env, registerResult, "")
	defer drain(completeResp)
	require.Equal(t, http.StatusCreated, completeResp.StatusCode)

	publicRegisterResp := env.do(t, http.MethodPost, "/api/v0/servers", map[string]any{
		"name":        "publicsrv",
		"description": "Public upstream MCP server",
		"transport":   "streamable_http",
		"url":         publicUpstream.server.URL + "/mcp",
	}, "")
	defer drain(publicRegisterResp)
	require.Equal(t, http.StatusCreated, publicRegisterResp.StatusCode)

	invokeResp := env.do(t, http.MethodPost, "/api/v0/tools/invoke", map[string]any{
		"name": "publicsrv__echo",
		"msg":  "hello public",
	}, "")
	defer drain(invokeResp)
	require.Equal(t, http.StatusOK, invokeResp.StatusCode)

	var invokeResult toolInvokeResult
	decodeJSON(t, invokeResp, &invokeResult)
	require.NotEmpty(t, invokeResult.Content)
	assert.Contains(t, invokeResult.Content[0].Text, "hello public")
	assert.Zero(t, publicUpstream.requestsWithAuth)
}
