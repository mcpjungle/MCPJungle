package mcp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func lazyTestService(t *testing.T) *MCPService {
	t.Helper()
	db := setupLazyTestDB(t)
	service := &MCPService{
		db:                         db,
		metrics:                    telemetry.NewNoopCustomMetrics(),
		mcpServerInitReqTimeoutSec: 5,
		lazyMcpProxyServer:         newLazyMCPServer(),
	}
	service.initializeLazyHandlers()
	return service
}

// setupLazyTestDB creates an in-memory database with every table the lazy
// handlers and the upstream session manager read from.
func setupLazyTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.McpServer{},
		&model.Tool{},
		&model.Prompt{},
		&model.UpstreamOAuthToken{},
		&model.UpstreamOAuthPendingSession{},
	))
	return db
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

func lazyResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	return text.Text
}

type lazyTestGroupResolver struct {
	tools []string
	err   error
	calls *int
}

func (r lazyTestGroupResolver) ResolveEffectiveTools(string) ([]string, error) {
	if r.calls != nil {
		(*r.calls)++
	}
	return r.tools, r.err
}

func lazyDevContext() context.Context {
	return context.WithValue(context.Background(), "mode", model.ModeDev)
}

func lazyEnterpriseContext(client *model.McpClient) context.Context {
	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	return context.WithValue(ctx, "client", client)
}

func TestLazyProxyServer_RegistersOnlyMetaTools(t *testing.T) {
	service := lazyTestService(t)
	tools := service.LazyMcpProxyServer().ListTools()
	assert.Len(t, tools, 3)
	assert.Contains(t, tools, metaToolListTools)
	assert.Contains(t, tools, metaToolDescribeTools)
	assert.Contains(t, tools, metaToolCallTool)
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
	assert.Contains(t, result.Instructions, metaToolListTools)
	assert.Contains(t, result.Instructions, metaToolDescribeTools)
	assert.Contains(t, result.Instructions, metaToolCallTool)
	assert.Contains(t, result.Instructions, "Never use a generic MCP context search")
}

func TestLazyDiscovery_ListToolsReturnsNamesAndDescriptionsOnly(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	beta := createLazyTestServer(t, service, "beta")
	disabled := createLazyTestServer(t, service, "gamma")
	require.NoError(t, service.db.Model(disabled).Update("enabled", false).Error)

	createLazyTestTool(t, service, alpha, "search", true)
	createLazyTestTool(t, service, alpha, "disabled", false)
	createLazyTestTool(t, service, beta, "lookup", true)

	result, err := service.lazyListToolsHandler(lazyDevContext(), lazyRequest(metaToolListTools, nil))
	require.NoError(t, err)

	var entries []map[string]any
	lazyResultJSON(t, result, &entries)
	require.Len(t, entries, 2, "disabled server and disabled tool must be filtered out")

	assert.Equal(t, "alpha__search", entries[0]["name"])
	assert.Equal(t, "search description", entries[0]["description"])
	assert.Equal(t, "beta__lookup", entries[1]["name"])
	assert.NotContains(t, entries[0], "input_schema", "list must not expose schemas")
	assert.NotContains(t, entries[0], "transport")
	assert.NotContains(t, entries[0], "enabled_tool_count")
}

func TestLazyDiscovery_ListToolsTruncatesDescriptions(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	createLazyTestTool(t, service, alpha, "search", true)
	longDescription := ""
	for i := 0; i < 300; i++ {
		longDescription += "x"
	}
	require.NoError(t, service.db.Model(&model.Tool{}).Where("server_id = ?", alpha.ID).
		Update("description", longDescription).Error)

	result, err := service.lazyListToolsHandler(lazyDevContext(), lazyRequest(metaToolListTools, nil))
	require.NoError(t, err)

	var entries []map[string]any
	lazyResultJSON(t, result, &entries)
	require.Len(t, entries, 1)
	assert.Len(t, []rune(entries[0]["description"].(string)), maxLazyDescriptionLength)
}

func TestLazyDiscovery_ListToolsFiltersByServer(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	beta := createLazyTestServer(t, service, "beta")
	createLazyTestTool(t, service, alpha, "search", true)
	createLazyTestTool(t, service, beta, "lookup", true)

	result, err := service.lazyListToolsHandler(lazyDevContext(), lazyRequest(metaToolListTools, map[string]any{"server": "alpha"}))
	require.NoError(t, err)

	var entries []map[string]any
	lazyResultJSON(t, result, &entries)
	require.Len(t, entries, 1)
	assert.Equal(t, "alpha__search", entries[0]["name"])
}

func TestLazyDiscovery_ListToolsEnterpriseAllowList(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	beta := createLazyTestServer(t, service, "beta")
	createLazyTestTool(t, service, alpha, "search", true)
	createLazyTestTool(t, service, beta, "lookup", true)

	ctx := lazyEnterpriseContext(&model.McpClient{
		Name:      "restricted-client",
		AllowList: datatypes.JSON(`["alpha"]`),
	})

	result, err := service.lazyListToolsHandler(ctx, lazyRequest(metaToolListTools, nil))
	require.NoError(t, err)

	var entries []map[string]any
	lazyResultJSON(t, result, &entries)
	require.Len(t, entries, 1)
	assert.Equal(t, "alpha__search", entries[0]["name"])
}

func TestLazyDiscovery_DescribeToolsReturnsSelectedSchemas(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	createLazyTestTool(t, service, alpha, "search", true)
	createLazyTestTool(t, service, alpha, "lookup", true)
	createLazyTestTool(t, service, alpha, "other", true)

	result, err := service.lazyDescribeToolsHandler(lazyDevContext(), lazyRequest(metaToolDescribeTools, map[string]any{
		"names": []string{"alpha__search", "alpha__lookup"},
	}))
	require.NoError(t, err)

	var tools []map[string]any
	lazyResultJSON(t, result, &tools)
	require.Len(t, tools, 2, "only the requested tools may be described")
	assert.Equal(t, "alpha__lookup", tools[0]["name"])
	assert.Equal(t, "alpha__search", tools[1]["name"])
	assert.NotEmpty(t, tools[0]["input_schema"])
	assert.NotEmpty(t, tools[0]["description"])
}

func TestLazyDiscovery_DescribeToolsAcceptsCommaSeparatedNames(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	createLazyTestTool(t, service, alpha, "search", true)

	result, err := service.lazyDescribeToolsHandler(lazyDevContext(), lazyRequest(metaToolDescribeTools, map[string]any{
		"names": "alpha__search",
	}))
	require.NoError(t, err)

	var tools []map[string]any
	lazyResultJSON(t, result, &tools)
	require.Len(t, tools, 1)
	assert.Equal(t, "alpha__search", tools[0]["name"])
}

func TestLazyDiscovery_DescribeToolsRejectsInvalidNames(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	createLazyTestTool(t, service, alpha, "search", true)

	tooMany := make([]string, 0, maxDescribeToolNames+1)
	for i := 0; i <= maxDescribeToolNames; i++ {
		tooMany = append(tooMany, "alpha__search")
	}

	cases := map[string]map[string]any{
		"missing":    {},
		"empty":      {"names": []string{}},
		"duplicate":  {"names": []string{"alpha__search", "alpha__search"}},
		"too many":   {"names": tooMany},
		"not a list": {"names": 42},
	}
	for name, arguments := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := service.lazyDescribeToolsHandler(lazyDevContext(), lazyRequest(metaToolDescribeTools, arguments))
			require.NoError(t, err)
			require.True(t, result.IsError)
		})
	}
}

func TestLazyDiscovery_DescribeToolsUnknownToolIsError(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	createLazyTestTool(t, service, alpha, "search", true)

	result, err := service.lazyDescribeToolsHandler(lazyDevContext(), lazyRequest(metaToolDescribeTools, map[string]any{
		"names": []string{"alpha__missing"},
	}))
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestLazyDiscovery_DescribeToolsEnterpriseAllowList(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	beta := createLazyTestServer(t, service, "beta")
	createLazyTestTool(t, service, alpha, "search", true)
	createLazyTestTool(t, service, beta, "lookup", true)

	ctx := lazyEnterpriseContext(&model.McpClient{
		Name:      "restricted-client",
		AllowList: datatypes.JSON(`["beta"]`),
	})

	denied, err := service.lazyDescribeToolsHandler(ctx, lazyRequest(metaToolDescribeTools, map[string]any{
		"names": []string{"alpha__search"},
	}))
	require.NoError(t, err)
	require.True(t, denied.IsError)

	allowed, err := service.lazyDescribeToolsHandler(ctx, lazyRequest(metaToolDescribeTools, map[string]any{
		"names": []string{"beta__lookup"},
	}))
	require.NoError(t, err)
	var tools []map[string]any
	lazyResultJSON(t, allowed, &tools)
	require.Len(t, tools, 1)
	assert.Equal(t, "beta__lookup", tools[0]["name"])
}

func TestLazyDiscovery_CallToolRejectsUnknownTool(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	createLazyTestTool(t, service, alpha, "search", true)

	result, err := service.lazyCallToolHandler(lazyDevContext(), lazyRequest(metaToolCallTool, map[string]any{
		"name": "alpha__missing",
	}))
	require.NoError(t, err)
	require.True(t, result.IsError)

	message := lazyResultText(t, result)
	assert.Contains(t, message, "alpha__search", "the nearest available tool must be suggested")
	assert.Contains(t, message, metaToolListTools)
}

func TestLazyDiscovery_CallToolRejectsNonCanonicalName(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	createLazyTestTool(t, service, alpha, "search", true)

	result, err := service.lazyCallToolHandler(lazyDevContext(), lazyRequest(metaToolCallTool, map[string]any{
		"name": "search",
	}))
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, lazyResultText(t, result), "alpha__search")
}

func TestLazyDiscovery_CallToolRequiresName(t *testing.T) {
	service := lazyTestService(t)

	result, err := service.lazyCallToolHandler(lazyDevContext(), lazyRequest(metaToolCallTool, nil))
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, lazyResultText(t, result), metaToolListTools)
}

func TestLazyDiscovery_CallToolEnterpriseAllowList(t *testing.T) {
	service := lazyTestService(t)
	upstreamCalls := 0
	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0")
	upstream.AddTool(
		mcp.NewTool("search", mcp.WithString("query")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			upstreamCalls++
			return mcp.NewToolResultText("called"), nil
		},
	)
	httpServer := httptest.NewServer(mcpserver.NewStreamableHTTPServer(upstream))
	defer httpServer.Close()

	serverModel, err := model.NewStreamableHTTPServer("alpha", "alpha description", httpServer.URL, "", nil, types.SessionModeStateless)
	require.NoError(t, err)
	require.NoError(t, service.db.Create(serverModel).Error)
	createLazyTestTool(t, service, serverModel, "search", true)

	ctx := lazyEnterpriseContext(&model.McpClient{
		Name:      "restricted-client",
		AllowList: datatypes.JSON(`["beta"]`),
	})

	result, err := service.lazyCallToolHandler(ctx, lazyRequest(metaToolCallTool, map[string]any{
		"name":      "alpha__search",
		"arguments": map[string]any{"query": "hello"},
	}))
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Zero(t, upstreamCalls, "a denied call must never reach the upstream server")
}

func TestLazyDiscovery_GroupAndEnterpriseAuthorization(t *testing.T) {
	service := lazyTestService(t)
	upstreamCalls := 0
	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0")
	upstream.AddTool(
		mcp.NewTool("search", mcp.WithString("query")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			upstreamCalls++
			return mcp.NewToolResultText("called"), nil
		},
	)
	httpServer := httptest.NewServer(mcpserver.NewStreamableHTTPServer(upstream))
	defer httpServer.Close()

	alpha, err := model.NewStreamableHTTPServer("alpha", "alpha description", httpServer.URL, "", nil, types.SessionModeStateless)
	require.NoError(t, err)
	require.NoError(t, service.db.Create(alpha).Error)
	createLazyTestTool(t, service, alpha, "allowed", true)
	createLazyTestTool(t, service, alpha, "secret", true)

	beta, err := model.NewStreamableHTTPServer("beta", "beta description", httpServer.URL, "", nil, types.SessionModeStateless)
	require.NoError(t, err)
	require.NoError(t, service.db.Create(beta).Error)
	createLazyTestTool(t, service, beta, "lookup", true)

	service.toolGroupService = lazyTestGroupResolver{tools: []string{"alpha__allowed", "beta__lookup"}}
	ctx := service.WithLazyGroup(lazyEnterpriseContext(&model.McpClient{
		Name:      "restricted-client",
		AllowList: datatypes.JSON(`["alpha"]`),
	}), "restricted-group")

	listResult, err := service.lazyListToolsHandler(ctx, lazyRequest(metaToolListTools, nil))
	require.NoError(t, err)
	var entries []map[string]any
	lazyResultJSON(t, listResult, &entries)
	require.Len(t, entries, 1)
	assert.Equal(t, "alpha__allowed", entries[0]["name"])

	for _, name := range []string{"alpha__secret", "beta__lookup"} {
		t.Run(name, func(t *testing.T) {
			describeResult, err := service.lazyDescribeToolsHandler(ctx, lazyRequest(metaToolDescribeTools, map[string]any{
				"names": []string{name},
			}))
			require.NoError(t, err)
			require.True(t, describeResult.IsError)

			callResult, err := service.lazyCallToolHandler(ctx, lazyRequest(metaToolCallTool, map[string]any{
				"name":      name,
				"arguments": map[string]any{"query": "hello"},
			}))
			require.NoError(t, err)
			require.True(t, callResult.IsError)
		})
	}
	assert.Zero(t, upstreamCalls, "denied calls must never reach the upstream server")
}

func TestLazyDiscovery_ResolvesToolGroupOncePerOperation(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	createLazyTestTool(t, service, alpha, "search", true)
	createLazyTestTool(t, service, alpha, "lookup", true)
	createLazyTestTool(t, service, alpha, "excluded", true)

	resolutions := 0
	service.toolGroupService = lazyTestGroupResolver{
		tools: []string{"alpha__search", "alpha__lookup"},
		calls: &resolutions,
	}
	ctx := service.WithLazyGroup(lazyDevContext(), "restricted-group")

	t.Run("describe multiple tools", func(t *testing.T) {
		resolutions = 0
		result, err := service.lazyDescribeToolsHandler(ctx, lazyRequest(metaToolDescribeTools, map[string]any{
			"names": []string{"alpha__search", "alpha__lookup"},
		}))
		require.NoError(t, err)
		require.False(t, result.IsError, lazyResultText(t, result))
		assert.Equal(t, 1, resolutions)
	})

	t.Run("list", func(t *testing.T) {
		resolutions = 0
		result, err := service.lazyListToolsHandler(ctx, lazyRequest(metaToolListTools, nil))
		require.NoError(t, err)
		require.False(t, result.IsError, lazyResultText(t, result))
		assert.Equal(t, 1, resolutions)
	})

	t.Run("denied call including suggestions", func(t *testing.T) {
		resolutions = 0
		result, err := service.lazyCallToolHandler(ctx, lazyRequest(metaToolCallTool, map[string]any{
			"name": "alpha__excluded",
		}))
		require.NoError(t, err)
		require.True(t, result.IsError)
		assert.Equal(t, 1, resolutions)
	})
}

func TestLazyDiscovery_CallToolInvokesUpstream(t *testing.T) {
	service := lazyTestService(t)
	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0")
	upstream.AddTool(
		mcp.NewTool("search", mcp.WithString("query")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, _ := req.GetArguments()["query"].(string)
			return mcp.NewToolResultText("upstream:" + query), nil
		},
	)
	httpServer := httptest.NewServer(mcpserver.NewStreamableHTTPServer(upstream))
	defer httpServer.Close()

	serverModel, err := model.NewStreamableHTTPServer("alpha", "alpha description", httpServer.URL, "", nil, types.SessionModeStateless)
	require.NoError(t, err)
	require.NoError(t, service.db.Create(serverModel).Error)
	createLazyTestTool(t, service, serverModel, "search", true)

	result, err := service.lazyCallToolHandler(lazyDevContext(), lazyRequest(metaToolCallTool, map[string]any{
		"name":      "alpha__search",
		"arguments": map[string]any{"query": "hello"},
	}))
	require.NoError(t, err)
	require.False(t, result.IsError, lazyResultText(t, result))
	assert.Contains(t, lazyResultText(t, result), "upstream:hello")
}

func TestLazyDiscovery_CallToolAcceptsJSONStringArguments(t *testing.T) {
	service := lazyTestService(t)
	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0")
	upstream.AddTool(
		mcp.NewTool("search", mcp.WithString("query")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, _ := req.GetArguments()["query"].(string)
			return mcp.NewToolResultText("upstream:" + query), nil
		},
	)
	httpServer := httptest.NewServer(mcpserver.NewStreamableHTTPServer(upstream))
	defer httpServer.Close()

	serverModel, err := model.NewStreamableHTTPServer("alpha", "alpha description", httpServer.URL, "", nil, types.SessionModeStateless)
	require.NoError(t, err)
	require.NoError(t, service.db.Create(serverModel).Error)
	createLazyTestTool(t, service, serverModel, "search", true)

	result, err := service.lazyCallToolHandler(lazyDevContext(), lazyRequest(metaToolCallTool, map[string]any{
		"name":      "alpha__search",
		"arguments": `{"query":"hello"}`,
	}))
	require.NoError(t, err)
	require.False(t, result.IsError, lazyResultText(t, result))
	assert.Contains(t, lazyResultText(t, result), "upstream:hello")
}

func TestLazyDiscovery_CallToolRejectsInvalidArguments(t *testing.T) {
	service := lazyTestService(t)
	alpha := createLazyTestServer(t, service, "alpha")
	createLazyTestTool(t, service, alpha, "search", true)

	result, err := service.lazyCallToolHandler(lazyDevContext(), lazyRequest(metaToolCallTool, map[string]any{
		"name":      "alpha__search",
		"arguments": "not-json",
	}))
	require.NoError(t, err)
	require.True(t, result.IsError)
}

func TestLazyDiscovery_ListDescribeAndCallShareOnePolicy(t *testing.T) {
	service := lazyTestService(t)
	upstreamCalls := 0
	upstream := mcpserver.NewMCPServer("Upstream", "0.1.0")
	for _, name := range []string{"allowed", "disabled", "lookup"} {
		upstream.AddTool(
			mcp.NewTool(name, mcp.WithString("query")),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				upstreamCalls++
				return mcp.NewToolResultText("called"), nil
			},
		)
	}
	httpServer := httptest.NewServer(mcpserver.NewStreamableHTTPServer(upstream))
	defer httpServer.Close()

	alpha, err := model.NewStreamableHTTPServer("alpha", "alpha description", httpServer.URL, "", nil, types.SessionModeStateless)
	require.NoError(t, err)
	require.NoError(t, service.db.Create(alpha).Error)
	createLazyTestTool(t, service, alpha, "allowed", true)
	createLazyTestTool(t, service, alpha, "disabled", false)

	beta, err := model.NewStreamableHTTPServer("beta", "beta description", httpServer.URL, "", nil, types.SessionModeStateless)
	require.NoError(t, err)
	require.NoError(t, service.db.Create(beta).Error)
	createLazyTestTool(t, service, beta, "lookup", true)

	cases := []struct {
		name    string
		tool    string
		visible bool
	}{
		{name: "enabled tool", tool: "alpha__allowed", visible: true},
		{name: "disabled tool", tool: "alpha__disabled", visible: false},
		{name: "unknown tool", tool: "alpha__missing", visible: false},
		{name: "tool of another server", tool: "beta__lookup", visible: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := lazyDevContext()

			listResult, err := service.lazyListToolsHandler(ctx, lazyRequest(metaToolListTools, nil))
			require.NoError(t, err)
			var entries []map[string]any
			lazyResultJSON(t, listResult, &entries)
			assert.Equal(t, tc.visible, containsTool(entries, tc.tool))

			describeResult, err := service.lazyDescribeToolsHandler(ctx, lazyRequest(metaToolDescribeTools, map[string]any{
				"names": []string{tc.tool},
			}))
			require.NoError(t, err)
			assert.Equal(t, tc.visible, !describeResult.IsError)

			callResult, err := service.lazyCallToolHandler(ctx, lazyRequest(metaToolCallTool, map[string]any{
				"name": tc.tool,
			}))
			require.NoError(t, err)
			assert.Equal(t, tc.visible, !callResult.IsError)
		})
	}
	assert.Equal(t, 2, upstreamCalls, "only the permitted tools may reach the upstream server")
}

func containsTool(entries []map[string]any, name string) bool {
	for _, entry := range entries {
		if entry["name"] == name {
			return true
		}
	}
	return false
}

func TestTruncateDescriptionKeepsMultiByteCharacters(t *testing.T) {
	description := ""
	for i := 0; i < 200; i++ {
		description += "é"
	}
	truncated := truncateDescription(description, maxLazyDescriptionLength)
	assert.Len(t, []rune(truncated), maxLazyDescriptionLength)
	assert.True(t, len(truncated) > maxLazyDescriptionLength, "multi-byte characters must not be cut in half")
}
