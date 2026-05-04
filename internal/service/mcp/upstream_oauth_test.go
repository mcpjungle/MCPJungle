package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	mcpgoclient "github.com/mark3labs/mcp-go/client"
	mcpgotransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/require"
)

type mockOAuthUpstream struct {
	server           *httptest.Server
	accessToken      string
	registerCalls    int
	lastRegisterBody map[string]any
}

func newMockOAuthUpstream(t *testing.T) *mockOAuthUpstream {
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

	mock := &mockOAuthUpstream{accessToken: "mock-access-token"}
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
		defer r.Body.Close()
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		mock.lastRegisterBody = body

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
		if r.Header.Get("Authorization") != "Bearer "+mock.accessToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}
		streamable.ServeHTTP(w, r)
	}))

	mock.server = httptest.NewServer(mux)
	t.Cleanup(mock.server.Close)
	return mock
}

func newOAuthCapableService(t *testing.T) (*MCPService, *testhelpers.TestDBSetup) {
	t.Helper()

	setup := testhelpers.SetupTestDB(t)
	service, err := NewMCPService(&ServiceConfig{
		DB:                      setup.DB,
		McpProxyServer:          mcpserver.NewMCPServer("test-proxy", "0.1.0"),
		SseMcpProxyServer:       mcpserver.NewMCPServer("test-proxy-sse", "0.1.0"),
		Metrics:                 telemetry.NewNoopCustomMetrics(),
		McpServerInitReqTimeout: 5,
	})
	require.NoError(t, err)
	t.Cleanup(service.Shutdown)

	return service, setup
}

func newOAuthHTTPServerModel(t *testing.T, upstreamURL string) *model.McpServer {
	t.Helper()

	srv, err := model.NewStreamableHTTPServer(
		"todoist",
		"OAuth upstream",
		upstreamURL,
		"",
		nil,
		types.SessionModeStateless,
	)
	require.NoError(t, err)
	return srv
}

func TestGenerateOAuthSessionID(t *testing.T) {
	t.Parallel()

	first, err := generateOAuthSessionID()
	testhelpers.AssertNoError(t, err)
	second, err := generateOAuthSessionID()
	testhelpers.AssertNoError(t, err)

	if len(first) != 64 {
		t.Fatalf("expected hex session ID length 64, got %d", len(first))
	}
	if first == second {
		t.Fatal("expected two generated session IDs to differ")
	}
}

func TestScopesJSONRoundTrip(t *testing.T) {
	t.Parallel()

	data := scopesToJSON([]string{"mcp.read", "tasks.read"})
	var raw []string
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("expected valid JSON array, got error: %v", err)
	}

	scopes, err := scopesFromJSON(data)
	testhelpers.AssertNoError(t, err)
	if len(scopes) != 2 || scopes[0] != "mcp.read" || scopes[1] != "tasks.read" {
		t.Fatalf("unexpected scopes round-trip result: %#v", scopes)
	}
}

func TestUpstreamOAuthTokenStore_SaveAndGetToken(t *testing.T) {
	t.Parallel()

	setup := testhelpers.SetupTestDB(t)

	record := &model.UpstreamOAuthToken{
		ServerName:   "todoist",
		Transport:    types.TransportStreamableHTTP,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "http://127.0.0.1:7777/oauth/callback",
		Scopes:       scopesToJSON([]string{"mcp.read"}),
	}
	testhelpers.AssertNoError(t, setup.DB.Create(record).Error)

	store := &upstreamOAuthTokenStore{
		db:         setup.DB,
		serverName: "todoist",
		transport:  types.TransportStreamableHTTP,
	}

	token := &mcpgotransport.Token{
		AccessToken:  "access-token",
		TokenType:    "Bearer",
		RefreshToken: "refresh-token",
		Scope:        "mcp.read",
		ExpiresAt:    time.Now().Add(5 * time.Minute).UTC(),
	}
	testhelpers.AssertNoError(t, store.SaveToken(context.Background(), token))

	saved, err := store.GetToken(context.Background())
	testhelpers.AssertNoError(t, err)
	if saved.AccessToken != token.AccessToken {
		t.Fatalf("expected access token %q, got %q", token.AccessToken, saved.AccessToken)
	}

	var persisted model.UpstreamOAuthToken
	testhelpers.AssertNoError(t, setup.DB.Where("server_name = ?", "todoist").First(&persisted).Error)
	if persisted.ClientID != "client-id" {
		t.Fatalf("expected existing client metadata to be preserved, got %q", persisted.ClientID)
	}
	if persisted.RefreshToken != "refresh-token" {
		t.Fatalf("expected refresh token to be updated, got %q", persisted.RefreshToken)
	}
}

func TestUpstreamOAuthDCRUnsupportedUserError(t *testing.T) {
	t.Parallel()

	err := upstreamOAuthDCRUnsupportedUserError()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "does not support dynamic client registration") {
		t.Fatalf("expected DCR limitation in error message, got %q", msg)
	}
	if !strings.Contains(msg, "oauth_client_id") || !strings.Contains(msg, "oauth_client_secret") {
		t.Fatalf("expected credential hint in error message, got %q", msg)
	}
}

func TestRegisterOAuthClientWithoutEmptyScope_OmitsScopeWhenUnset(t *testing.T) {
	t.Parallel()

	upstream := newMockOAuthUpstream(t)

	oauthClient, err := mcpgoclient.NewOAuthStreamableHttpClient(
		upstream.server.URL+"/mcp",
		mcpgoclient.OAuthConfig{
			RedirectURI: upstream.server.URL + "/callback",
			TokenStore:  mcpgoclient.NewMemoryTokenStore(),
			PKCEEnabled: true,
		},
	)
	require.NoError(t, err)
	defer oauthClient.Close()

	_, err = initializeHTTPClient(context.Background(), oauthClient, upstream.server.URL+"/mcp", 5)
	require.Error(t, err)

	var oauthErr *mcpgoclient.OAuthAuthorizationRequiredError
	require.True(t, errors.As(err, &oauthErr))
	require.NotNil(t, oauthErr.Handler)

	clientID, clientSecret, err := registerOAuthClientWithoutEmptyScope(
		context.Background(),
		oauthErr.Handler,
		&types.RegisterServerInput{
			OAuthRedirectURI: upstream.server.URL + "/callback",
		},
		"mcpjungle-test",
	)
	require.NoError(t, err)
	require.Equal(t, "mock-client-id", clientID)
	require.Empty(t, clientSecret)
	require.Equal(t, 1, upstream.registerCalls)
	if _, exists := upstream.lastRegisterBody["scope"]; exists {
		t.Fatalf("expected scope to be omitted from DCR payload, got %#v", upstream.lastRegisterBody["scope"])
	}
}

func TestBootstrapUpstreamOAuth_UsesConfiguredClientCredentialsWithoutDCR(t *testing.T) {
	t.Parallel()

	service, setup := newOAuthCapableService(t)
	upstream := newMockOAuthUpstream(t)
	server := newOAuthHTTPServerModel(t, upstream.server.URL+"/mcp")

	input := &types.RegisterServerInput{
		Name:              "todoist",
		Transport:         string(types.TransportStreamableHTTP),
		URL:               upstream.server.URL + "/mcp",
		OAuthRedirectURI:  "http://127.0.0.1:9999/oauth/callback",
		OAuthClientID:     "provider-client-id",
		OAuthClientSecret: "provider-client-secret",
	}

	err := service.RegisterMcpServerWithOAuthSupport(context.Background(), input, server, false, "tester")
	require.Error(t, err)

	var pendingErr *UpstreamOAuthAuthorizationPendingError
	require.True(t, errors.As(err, &pendingErr))
	require.Equal(t, 0, upstream.registerCalls)

	var session model.UpstreamOAuthPendingSession
	require.NoError(t, setup.DB.Where("session_id = ?", pendingErr.SessionID).First(&session).Error)
	require.Equal(t, input.OAuthClientID, session.ClientID)
	require.Equal(t, input.OAuthClientSecret, session.ClientSecret)
}

func TestBootstrapUpstreamOAuth_ReplacesExistingPendingSessionWithHardDelete(t *testing.T) {
	t.Parallel()

	service, setup := newOAuthCapableService(t)
	upstream := newMockOAuthUpstream(t)
	server := newOAuthHTTPServerModel(t, upstream.server.URL+"/mcp")

	stale := &model.UpstreamOAuthPendingSession{
		SessionID:    "stale-session",
		ServerName:   "todoist",
		Transport:    types.TransportStreamableHTTP,
		ServerInput:  scopesToJSON(nil),
		RedirectURI:  "http://127.0.0.1:8888/oauth/callback",
		ClientID:     "old-client",
		State:        "stale-state",
		CodeVerifier: "stale-verifier",
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}
	require.NoError(t, setup.DB.Create(stale).Error)

	input := &types.RegisterServerInput{
		Name:             "todoist",
		Transport:        string(types.TransportStreamableHTTP),
		URL:              upstream.server.URL + "/mcp",
		OAuthRedirectURI: "http://127.0.0.1:9999/oauth/callback",
		OAuthScopes:      []string{"mcp.read"},
	}

	err := service.RegisterMcpServerWithOAuthSupport(context.Background(), input, server, false, "tester")
	require.Error(t, err)

	var pendingErr *UpstreamOAuthAuthorizationPendingError
	require.True(t, errors.As(err, &pendingErr))

	var pendingSessions []model.UpstreamOAuthPendingSession
	require.NoError(t, setup.DB.Unscoped().Where("server_name = ?", "todoist").Find(&pendingSessions).Error)
	require.Len(t, pendingSessions, 1)
	require.Equal(t, pendingErr.SessionID, pendingSessions[0].SessionID)
}

func TestCompleteUpstreamOAuthSession_HardDeletesPendingSessionAfterSuccess(t *testing.T) {
	t.Parallel()

	service, setup := newOAuthCapableService(t)
	upstream := newMockOAuthUpstream(t)

	input := types.RegisterServerInput{
		Name:             "todoist",
		Description:      "OAuth upstream",
		Transport:        string(types.TransportStreamableHTTP),
		URL:              upstream.server.URL + "/mcp",
		OAuthRedirectURI: "http://127.0.0.1:9999/oauth/callback",
		OAuthClientID:    "mock-client-id",
		OAuthScopes:      []string{"mcp.read"},
	}
	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	session := &model.UpstreamOAuthPendingSession{
		SessionID:    "session-123",
		ServerName:   "todoist",
		Transport:    types.TransportStreamableHTTP,
		ServerInput:  inputJSON,
		RedirectURI:  input.OAuthRedirectURI,
		ClientID:     input.OAuthClientID,
		Scopes:       scopesToJSON(input.OAuthScopes),
		State:        "expected-state",
		CodeVerifier: "test-code-verifier",
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}
	require.NoError(t, setup.DB.Create(session).Error)

	server, err := service.CompleteUpstreamOAuthSession(
		context.Background(),
		session.SessionID,
		"mock-auth-code",
		session.State,
	)
	require.NoError(t, err)
	require.Equal(t, "todoist", server.Name)

	var pendingCount int64
	require.NoError(t, setup.DB.Unscoped().Model(&model.UpstreamOAuthPendingSession{}).Where("session_id = ?", session.SessionID).Count(&pendingCount).Error)
	require.Zero(t, pendingCount)
}

func TestDeregisterMcpServer_HardDeletesUpstreamOAuthState(t *testing.T) {
	t.Parallel()

	service, setup := newOAuthCapableService(t)

	serverModel := newOAuthHTTPServerModel(t, "https://example.com/mcp")
	require.NoError(t, setup.DB.Create(serverModel).Error)

	token := &model.UpstreamOAuthToken{
		ServerName: "todoist",
		Transport:  types.TransportStreamableHTTP,
		ClientID:   "client-id",
	}
	pending := &model.UpstreamOAuthPendingSession{
		SessionID:    "pending-123",
		ServerName:   "todoist",
		Transport:    types.TransportStreamableHTTP,
		ServerInput:  scopesToJSON(nil),
		State:        "state",
		CodeVerifier: "verifier",
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}
	require.NoError(t, setup.DB.Create(token).Error)
	require.NoError(t, setup.DB.Create(pending).Error)

	require.NoError(t, service.DeregisterMcpServer("todoist"))

	var tokenCount int64
	require.NoError(t, setup.DB.Unscoped().Model(&model.UpstreamOAuthToken{}).Where("server_name = ?", "todoist").Count(&tokenCount).Error)
	require.Zero(t, tokenCount)

	var pendingCount int64
	require.NoError(t, setup.DB.Unscoped().Model(&model.UpstreamOAuthPendingSession{}).Where("server_name = ?", "todoist").Count(&pendingCount).Error)
	require.Zero(t, pendingCount)
}
