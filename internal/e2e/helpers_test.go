// Package e2e contains end-to-end integration tests for MCPJungle against
// @modelcontextprotocol/server-everything.
//
// Tests spin up a full MCPJungle HTTP server backed by an in-memory SQLite
// database, register server-everything as a stdio upstream, then exercise every
// major API surface:
//   - Global tools: list, get, invoke
//   - Global prompts: list, get, render (simple and complex)
//   - Tool groups: CRUD, effective-tools, included-servers, excluded-tools
//   - Tool/prompt operations scoped to a tool group
//   - Dev mode vs Enterprise mode (auth, permissions, enterprise-only endpoints)
//   - MCP proxy client-token access control
package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/api"
	"github.com/mcpjungle/mcpjungle/internal/migrations"
	"github.com/mcpjungle/mcpjungle/internal/model"
	configSvc "github.com/mcpjungle/mcpjungle/internal/service/config"
	mcpSvc "github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/mcpjungle/mcpjungle/internal/service/mcpclient"
	"github.com/mcpjungle/mcpjungle/internal/service/toolgroup"
	userSvc "github.com/mcpjungle/mcpjungle/internal/service/user"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// -----------------------------------------------------------------------
// Shared response types
// -----------------------------------------------------------------------

// toolInvokeResult is the JSON response from POST /api/v0/tools/invoke.
type toolInvokeResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// renderedPromptResult is the JSON response from POST /api/v0/prompts/render.
type renderedPromptResult struct {
	Messages []struct {
		Role    string `json:"role"`
		Content struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"messages"`
}

// -----------------------------------------------------------------------
// Test environment
// -----------------------------------------------------------------------

// e2eEnv holds a running MCPJungle httptest server and associated tokens.
type e2eEnv struct {
	ts         *httptest.Server
	adminToken string // populated only in enterprise mode
	userToken  string // populated only in enterprise mode (regular user)
}

// do makes an HTTP request against the test server and returns the raw response.
// The caller is responsible for closing resp.Body (decodeJSON does it automatically).
func (e *e2eEnv) do(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.ts.URL+path, reqBody)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// drain closes an HTTP response body without reading it.
func drain(r *http.Response) { r.Body.Close() }

// decodeJSON decodes the JSON response body into target.
func decodeJSON(t *testing.T, r *http.Response, target any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(r.Body).Decode(target))
}

// setupE2EServer spins up a full MCPJungle HTTP server backed by an in-memory
// SQLite DB, initialised in the requested mode.
// In enterprise mode, env.adminToken and env.userToken are set.
// The server is shut down via t.Cleanup.
func setupE2EServer(t *testing.T, mode model.ServerMode) *e2eEnv {
	t.Helper()
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not found in PATH – skipping server-everything end-to-end tests")
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migrations.Migrate(db))

	mcpProxy := server.NewMCPServer("MCPJungle", "0.0.1",
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithToolFilter(mcpSvc.ProxyToolFilter),
	)
	sseMcpProxy := server.NewMCPServer("MCPJungle SSE", "0.0.1",
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithToolFilter(mcpSvc.ProxyToolFilter),
	)

	mcpService, err := mcpSvc.NewMCPService(&mcpSvc.ServiceConfig{
		DB:                      db,
		McpProxyServer:          mcpProxy,
		SseMcpProxyServer:       sseMcpProxy,
		Metrics:                 telemetry.NewNoopCustomMetrics(),
		McpServerInitReqTimeout: 30,
	})
	require.NoError(t, err)

	cfgSvc := configSvc.NewServerConfigService(db)
	usrSvc := userSvc.NewUserService(db)
	tgSvc, err := toolgroup.NewToolGroupService(db, mcpService)
	require.NoError(t, err)

	apiServer, err := api.NewServer(&api.ServerOptions{
		MCPProxyServer:    mcpProxy,
		SseMcpProxyServer: sseMcpProxy,
		MCPService:        mcpService,
		MCPClientService:  mcpclient.NewMCPClientService(db),
		ConfigService:     cfgSvc,
		UserService:       usrSvc,
		ToolGroupService:  tgSvc,
		Metrics:           telemetry.NewNoopCustomMetrics(),
	})
	require.NoError(t, err)

	env := &e2eEnv{}

	switch mode {
	case model.ModeDev:
		require.NoError(t, apiServer.InitDev())
	case model.ModeEnterprise:
		_, err = cfgSvc.Init(model.ModeEnterprise)
		require.NoError(t, err)
		adminUser, err := usrSvc.CreateAdminUser()
		require.NoError(t, err)
		env.adminToken = adminUser.AccessToken
		regularUser, err := usrSvc.CreateUser(&model.User{Username: "regularuser"})
		require.NoError(t, err)
		env.userToken = regularUser.AccessToken
	default:
		t.Fatalf("unsupported server mode: %s", mode)
	}

	ts := httptest.NewServer(apiServer.Router())
	t.Cleanup(ts.Close)
	env.ts = ts

	return env
}

// registerEverythingServer registers @modelcontextprotocol/server-everything
// as a stdio upstream named "everything" via the REST API.
func registerEverythingServer(t *testing.T, env *e2eEnv, token string) {
	t.Helper()
	registerEverythingServerAs(t, env, "everything", token)
}

// registerEverythingServerAs registers @modelcontextprotocol/server-everything
// under a custom name, allowing multiple instances with different names.
func registerEverythingServerAs(t *testing.T, env *e2eEnv, name string, token string) {
	t.Helper()
	body := map[string]any{
		"name":        name,
		"description": "MCP server-everything integration test server",
		"transport":   "stdio",
		"command":     "npx",
		"args":        []string{"-y", "@modelcontextprotocol/server-everything", "stdio"},
	}
	resp := env.do(t, http.MethodPost, "/api/v0/servers", body, token)
	defer drain(resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "register server-everything as %q", name)
}

// toolNames extracts the "name" field from a slice of JSON objects.
func toolNames(tools []map[string]any) []string {
	names := make([]string, 0, len(tools))
	for _, tl := range tools {
		if n, ok := tl["name"].(string); ok {
			names = append(names, n)
		}
	}
	return names
}

// promptNames extracts the "name" field from a slice of JSON prompt objects.
func promptNames(prompts []map[string]any) []string {
	names := make([]string, 0, len(prompts))
	for _, p := range prompts {
		if n, ok := p["name"].(string); ok {
			names = append(names, n)
		}
	}
	return names
}

// createMcpClient creates an MCP client with the given allow list and returns its access token.
func createMcpClient(t *testing.T, env *e2eEnv, name string, allowList []string) string {
	t.Helper()
	resp := env.do(t, http.MethodPost, "/api/v0/clients", map[string]any{
		"name":       name,
		"allow_list": allowList,
	}, env.adminToken)
	defer drain(resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "create MCP client %q", name)
	var mcpClient map[string]any
	decodeJSON(t, resp, &mcpClient)
	token, ok := mcpClient["access_token"].(string)
	require.True(t, ok, "access_token must be a string")
	require.NotEmpty(t, token)
	return token
}

// newMCPProxyClient creates an initialized StreamableHTTP MCP client that
// connects to the global /mcp endpoint using the provided client token.
func newMCPProxyClient(t *testing.T, env *e2eEnv, clientToken string) *client.Client {
	t.Helper()
	opts := []transport.StreamableHTTPCOption{}
	if clientToken != "" {
		opts = append(opts, transport.WithHTTPHeaders(map[string]string{
			"Authorization": "Bearer " + clientToken,
		}))
	}
	c, err := client.NewStreamableHttpClient(env.ts.URL+"/mcp", opts...)
	require.NoError(t, err)
	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "e2e-test-client",
				Version: "1.0.0",
			},
		},
	})
	require.NoError(t, err)
	return c
}

// newGroupMCPClient creates an initialized StreamableHTTP MCP client that
// connects directly to a tool group's own MCP endpoint at /v0/groups/:name/mcp.
// This endpoint exposes ONLY the tools registered for that group.
func newGroupMCPClient(t *testing.T, env *e2eEnv, groupName string, token string) *client.Client {
	t.Helper()
	opts := []transport.StreamableHTTPCOption{}
	if token != "" {
		opts = append(opts, transport.WithHTTPHeaders(map[string]string{
			"Authorization": "Bearer " + token,
		}))
	}
	c, err := client.NewStreamableHttpClient(
		env.ts.URL+"/v0/groups/"+groupName+"/mcp",
		opts...,
	)
	require.NoError(t, err)
	_, err = c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "e2e-test-client",
				Version: "1.0.0",
			},
		},
	})
	require.NoError(t, err)
	return c
}
