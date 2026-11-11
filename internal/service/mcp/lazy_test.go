package mcp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func lazyTestService(t *testing.T) *MCPService {
	t.Helper()
	db := setupTestDBWithTools(t)
	service := &MCPService{
		db:                 db,
		metrics:            telemetry.NewNoopCustomMetrics(),
		lazyMcpProxyServer: newLazyMCPServer(),
	}
	service.initializeLazyHandlers()
	return service
}

func createLazyTestServer(t *testing.T, service *MCPService, name string) *model.McpServer {
	t.Helper()
	serverModel, err := model.NewStdioServer(name, name+" description", "echo", nil, nil, "")
	require.NoError(t, err)
	require.NoError(t, service.db.Create(serverModel).Error)
	return serverModel
}

func createLazyTestTool(t *testing.T, service *MCPService, serverModel *model.McpServer, name string, enabled bool) {
	t.Helper()
	schema, err := json.Marshal(map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}}})
	require.NoError(t, err)
	tool := &model.Tool{ServerID: serverModel.ID, Name: name, Description: name + " description", InputSchema: schema, Enabled: true}
	require.NoError(t, service.db.Create(tool).Error)
	require.NoError(t, service.db.Model(tool).Update("enabled", enabled).Error)
}

func lazyRequest(name string, arguments map[string]any) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = arguments
	return req
}

func lazyResultJSON(t *testing.T, result *mcp.CallToolResult, destination any) {
	t.Helper()
	require.False(t, result.IsError)
	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	require.NoError(t, json.Unmarshal([]byte(text.Text), destination))
}

type lazyTestSession struct {
	id            string
	initialized   bool
	notifications chan mcp.JSONRPCNotification
	tools         map[string]mcpserver.ServerTool
	mu            sync.RWMutex
}

func (s *lazyTestSession) Initialize()       { s.initialized = true }
func (s *lazyTestSession) Initialized() bool { return s.initialized }
func (s *lazyTestSession) NotificationChannel() chan<- mcp.JSONRPCNotification {
	return s.notifications
}

func (s *lazyTestSession) SessionID() string { return s.id }

func (s *lazyTestSession) GetSessionTools() map[string]mcpserver.ServerTool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]mcpserver.ServerTool, len(s.tools))
	for name, tool := range s.tools {
		result[name] = tool
	}
	return result
}

func (s *lazyTestSession) SetSessionTools(tools map[string]mcpserver.ServerTool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = tools
}

func lazyActivationContext(t *testing.T, service *MCPService, ctx context.Context) (context.Context, *lazyTestSession) {
	t.Helper()
	session := &lazyTestSession{
		id:            t.Name(),
		notifications: make(chan mcp.JSONRPCNotification, 4),
		tools:         make(map[string]mcpserver.ServerTool),
	}
	require.NoError(t, service.LazyMcpProxyServer().RegisterSession(ctx, session))
	session.Initialize()
	t.Cleanup(func() { service.LazyMcpProxyServer().UnregisterSession(ctx, session.id) })
	return service.LazyMcpProxyServer().WithContext(ctx, session), session
}

// TestLazyProxyServer_RegistersOnlyMetaTools pins the public surface of the
// lazy proxy: an agent must only see `mcpjungle__list_servers`,
// `mcpjungle__show_server_tools`, and the `mcpjungle__invoke` fallback for
// static clients. Upstream tools and the old `mcpjungle__list_tools`
// meta-tool must not leak through.
func TestLazyProxyServer_RegistersOnlyMetaTools(t *testing.T) {
	service := lazyTestService(t)
	tools := service.LazyMcpProxyServer().ListTools()
	assert.Len(t, tools, 3)
	assert.Contains(t, tools, metaToolListServers)
	assert.Contains(t, tools, metaToolShowServerTools)
	assert.Contains(t, tools, metaToolInvoke)
	assert.NotContains(t, tools, "mcpjungle__list_tools")
}

func TestLazyProxyServer_InitializeExplainsDiscoveryWorkflow(t *testing.T) {
	service := lazyTestService(t)
	client, err := mcpclient.NewInProcessClient(service.LazyMcpProxyServer())
	require.NoError(t, err)
	require.NoError(t, client.Start(context.Background()))
	t.Cleanup(func() { _ = client.Close() })

	result, err := client.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "test-client", Version: "1.0.0"},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, result.Instructions, metaToolListServers)
	assert.Contains(t, result.Instructions, metaToolShowServerTools)
	assert.Contains(t, result.Instructions, "Never use a generic MCP context search")
}

// TestLazyDiscovery_ListServersReturnsEveryEnabledServer verifies the happy
// path: in dev mode every enabled server is listed with its name,
// description, transport, and the count of enabled tools.
func TestLazyDiscovery_ListServersReturnsEveryEnabledServer(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	beta := createLazyTestServer(t, service, "beta")
	disabled := createLazyTestServer(t, service, "gamma")
	require.NoError(t, service.db.Model(disabled).Update("enabled", false).Error)

	createLazyTestTool(t, service, alpha, "search", true)
	createLazyTestTool(t, service, alpha, "disabled", false)
	createLazyTestTool(t, service, beta, "lookup", true)

	ctx := context.WithValue(context.Background(), "mode", model.ModeDev)
	result, err := service.lazyListServersHandler(ctx, lazyRequest(metaToolListServers, nil))
	require.NoError(t, err)

	var entries []map[string]any
	lazyResultJSON(t, result, &entries)
	require.Len(t, entries, 2, "disabled server must be filtered out")

	assert.Equal(t, []string{"alpha", "beta"}, []string{entries[0]["name"].(string), entries[1]["name"].(string)})
	assert.Equal(t, "alpha description", entries[0]["description"])
	assert.Equal(t, "beta description", entries[1]["description"])
	assert.Equal(t, "stdio", entries[0]["transport"])
	assert.Equal(t, "stdio", entries[1]["transport"])

	assert.EqualValues(t, 1, entries[0]["enabled_tool_count"], "disabled tool must not be counted")
	assert.EqualValues(t, 1, entries[1]["enabled_tool_count"])
}

// TestLazyDiscovery_ShowServerToolsRequiresServerArg guarantees we cannot
// accidentally regress to the old behaviour of dumping every registered tool.
// Without `server` the handler must reject the call.
func TestLazyDiscovery_ShowServerToolsRequiresServerArg(t *testing.T) {
	service := lazyTestService(t)
	createLazyTestServer(t, service, "alpha")
	ctx := context.WithValue(context.Background(), "mode", model.ModeDev)

	result, err := service.lazyShowServerToolsHandler(ctx, lazyRequest(metaToolShowServerTools, nil))
	require.NoError(t, err)
	require.True(t, result.IsError, "missing server arg must produce an error result")
}

// TestLazyDiscovery_ShowServerToolsReturnsOnlyRequestedServer confirms the
// scoped lookup returns the canonical tool list for one server and ignores
// others, plus honours the tool's enabled flag.
func TestLazyDiscovery_ShowServerToolsReturnsOnlyRequestedServer(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	beta := createLazyTestServer(t, service, "beta")
	createLazyTestTool(t, service, alpha, "search", true)
	createLazyTestTool(t, service, alpha, "disabled", false)
	createLazyTestTool(t, service, beta, "lookup", true)
	ctx := context.WithValue(context.Background(), "mode", model.ModeDev)
	ctx, _ = lazyActivationContext(t, service, ctx)

	result, err := service.lazyShowServerToolsHandler(ctx, lazyRequest(metaToolShowServerTools, map[string]any{"server": "alpha"}))
	require.NoError(t, err)
	var tools []map[string]any
	lazyResultJSON(t, result, &tools)

	require.Len(t, tools, 1, "only alpha__search must be returned; beta must not leak")
	assert.Equal(t, "alpha__search", tools[0]["name"])
	assert.NotEmpty(t, tools[0]["input_schema"])
	assert.NotEmpty(t, tools[0]["description"])
}

// TestLazyDiscovery_ShowServerToolsUnknownServerReturnsError covers the case
// where the requested server does not exist at all.
func TestLazyDiscovery_ShowServerToolsUnknownServerReturnsError(t *testing.T) {
	service := lazyTestService(t)
	ctx := context.WithValue(context.Background(), "mode", model.ModeDev)

	result, err := service.lazyShowServerToolsHandler(ctx, lazyRequest(metaToolShowServerTools, map[string]any{"server": "missing"}))
	require.NoError(t, err)
	require.True(t, result.IsError)
}

// TestLazyDiscovery_ListServersEnterpriseAllowList proves the handler honours
// the enterprise client allow list: an unlisted server must not be returned.
func TestLazyDiscovery_ListServersEnterpriseAllowList(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	beta := createLazyTestServer(t, service, "beta")
	createLazyTestTool(t, service, alpha, "search", true)
	createLazyTestTool(t, service, beta, "lookup", true)

	client := &model.McpClient{
		Name:      "restricted-client",
		AllowList: datatypes.JSON(`["alpha"]`),
	}
	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = context.WithValue(ctx, "client", client)

	result, err := service.lazyListServersHandler(ctx, lazyRequest(metaToolListServers, nil))
	require.NoError(t, err)
	var entries []map[string]any
	lazyResultJSON(t, result, &entries)
	require.Len(t, entries, 1)
	assert.Equal(t, "alpha", entries[0]["name"])
}

// TestLazyDiscovery_ShowServerToolsEnterpriseAllowList mirrors the list
// check for the per-server lookup.
func TestLazyDiscovery_ShowServerToolsEnterpriseAllowList(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	beta := createLazyTestServer(t, service, "beta")
	createLazyTestTool(t, service, alpha, "search", true)
	createLazyTestTool(t, service, beta, "lookup", true)

	client := &model.McpClient{
		Name:      "restricted-client",
		AllowList: datatypes.JSON(`["beta"]`),
	}
	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = context.WithValue(ctx, "client", client)
	ctx, _ = lazyActivationContext(t, service, ctx)

	// alpha is denied
	denied, err := service.lazyShowServerToolsHandler(ctx, lazyRequest(metaToolShowServerTools, map[string]any{"server": "alpha"}))
	require.NoError(t, err)
	require.True(t, denied.IsError)

	// beta is allowed
	allowed, err := service.lazyShowServerToolsHandler(ctx, lazyRequest(metaToolShowServerTools, map[string]any{"server": "beta"}))
	require.NoError(t, err)
	var tools []map[string]any
	lazyResultJSON(t, allowed, &tools)
	require.Len(t, tools, 1)
	assert.Equal(t, "beta__lookup", tools[0]["name"])
}

func TestLazyDiscovery_ShowServerToolsActivatesOnlyCurrentSession(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	beta := createLazyTestServer(t, service, "beta")
	createLazyTestTool(t, service, alpha, "search", true)
	createLazyTestTool(t, service, beta, "lookup", true)

	ctx := context.WithValue(context.Background(), "mode", model.ModeDev)
	ctx, session := lazyActivationContext(t, service, ctx)
	result, err := service.lazyShowServerToolsHandler(ctx, lazyRequest(metaToolShowServerTools, map[string]any{"server": "alpha"}))
	require.NoError(t, err)
	require.False(t, result.IsError)

	tools := session.GetSessionTools()
	require.Contains(t, tools, "alpha__search")
	assert.NotContains(t, tools, "beta__lookup")
	assert.Equal(t, "alpha__search", tools["alpha__search"].Tool.Name)
	require.Equal(t, "notifications/tools/list_changed", (<-session.notifications).Method)

	result, err = service.lazyShowServerToolsHandler(ctx, lazyRequest(metaToolShowServerTools, map[string]any{"server": "alpha"}))
	require.NoError(t, err)
	require.False(t, result.IsError)
	assert.Len(t, session.GetSessionTools(), 1, "re-activation must not duplicate tools")

	result, err = service.lazyShowServerToolsHandler(ctx, lazyRequest(metaToolShowServerTools, map[string]any{"server": "beta"}))
	require.NoError(t, err)
	require.False(t, result.IsError)
	assert.Contains(t, session.GetSessionTools(), "beta__lookup")
}

func TestLazyDiscovery_ShowServerToolsRequiresActiveSession(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	createLazyTestTool(t, service, alpha, "search", true)
	ctx := context.WithValue(context.Background(), "mode", model.ModeDev)

	result, err := service.lazyShowServerToolsHandler(ctx, lazyRequest(metaToolShowServerTools, map[string]any{"server": "alpha"}))
	require.NoError(t, err)
	require.True(t, result.IsError)
}
