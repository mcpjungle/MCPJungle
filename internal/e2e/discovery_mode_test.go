package e2e_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newDiscoveryMCPClient creates an initialized StreamableHTTP MCP client that
// connects to the discovery-mode MCP endpoint.
func newDiscoveryMCPClient(
	t *testing.T, env *e2eEnv, clientToken string,
) (*client.Client, *mcp.InitializeResult) {
	t.Helper()

	opts := []transport.StreamableHTTPCOption{}
	if clientToken != "" {
		opts = append(opts, transport.WithHTTPHeaders(map[string]string{
			"Authorization": "Bearer " + clientToken,
		}))
	}
	c, err := client.NewStreamableHttpClient(env.baseURL+"/mcp?mode=discovery", opts...)
	require.NoError(t, err)

	initResult, err := c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "e2e-discovery-client", Version: "1.0.0"},
		},
	})
	require.NoError(t, err)
	return c, initResult
}

// discoveryMetaToolNames is the exact tool surface discovery mode must expose.
var discoveryMetaToolNames = []string{"discover_tools", "get_tool_schema", "execute_tool"}

// discoveryToolText extracts the text payload of a tool result.
func discoveryToolText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
	text, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok, "result content is %T, want mcp.TextContent", result.Content[0])
	return text.Text
}

func callDiscoveryTool(
	t *testing.T, c *client.Client, name string, args map[string]any,
) *mcp.CallToolResult {
	t.Helper()
	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: name, Arguments: args},
	})
	require.NoError(t, err)
	return result
}

// listMCPToolNames lists tool names through the MCP protocol.
func listMCPToolNames(t *testing.T, c *client.Client) []string {
	t.Helper()
	result, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	return names
}

func TestE2E_DiscoveryMode_ToolSurfaceAndFlow(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	c, initResult := newDiscoveryMCPClient(t, env, "")

	t.Run("handshake carries the discovery preamble", func(t *testing.T) {
		assert.Contains(t, initResult.Instructions, "on-demand tool discovery",
			"handshake instructions must describe the discovery workflow")
	})

	t.Run("tools list exposes only the meta tools", func(t *testing.T) {
		assert.ElementsMatch(t, discoveryMetaToolNames, listMCPToolNames(t, c))
	})

	t.Run("discover_tools lists live servers", func(t *testing.T) {
		result := callDiscoveryTool(t, c, "discover_tools", map[string]any{})
		require.False(t, result.IsError, "discover_tools failed: %s", discoveryToolText(t, result))

		var payload struct {
			Domains []struct {
				Name      string `json:"name"`
				ToolCount int64  `json:"tool_count"`
			} `json:"domains"`
			TotalTools int64 `json:"total_tools"`
		}
		require.NoError(t, json.Unmarshal([]byte(discoveryToolText(t, result)), &payload))
		require.Len(t, payload.Domains, 1)
		assert.Equal(t, "everything", payload.Domains[0].Name)
		assert.Greater(t, payload.Domains[0].ToolCount, int64(0))
		assert.Equal(t, payload.Domains[0].ToolCount, payload.TotalTools)
	})

	t.Run("discover_tools lists one server's signatures", func(t *testing.T) {
		result := callDiscoveryTool(t, c, "discover_tools", map[string]any{"domain": "everything"})
		require.False(t, result.IsError, "discover_tools(domain) failed: %s", discoveryToolText(t, result))

		text := discoveryToolText(t, result)
		assert.Contains(t, text, "everything__echo(")
		assert.Contains(t, text, "-> any")
	})

	t.Run("unknown domain reports domain_not_found", func(t *testing.T) {
		result := callDiscoveryTool(t, c, "discover_tools", map[string]any{"domain": "nope"})
		require.True(t, result.IsError)

		var payload struct {
			Code    string `json:"code"`
			Details struct {
				AvailableDomains []string `json:"available_domains"`
			} `json:"details"`
		}
		require.NoError(t, json.Unmarshal([]byte(discoveryToolText(t, result)), &payload))
		assert.Equal(t, "domain_not_found", payload.Code)
		assert.Contains(t, payload.Details.AvailableDomains, "everything")
	})

	t.Run("get_tool_schema returns the input schema", func(t *testing.T) {
		result := callDiscoveryTool(t, c, "get_tool_schema", map[string]any{"tool_name": "everything__echo"})
		require.False(t, result.IsError, "get_tool_schema failed: %s", discoveryToolText(t, result))

		var payload struct {
			Name       string         `json:"name"`
			Domain     string         `json:"domain"`
			Parameters map[string]any `json:"parameters"`
		}
		require.NoError(t, json.Unmarshal([]byte(discoveryToolText(t, result)), &payload))
		assert.Equal(t, "everything__echo", payload.Name)
		assert.Equal(t, "everything", payload.Domain)
		assert.NotEmpty(t, payload.Parameters)
	})

	t.Run("unknown tool suggests a similar name", func(t *testing.T) {
		result := callDiscoveryTool(t, c, "get_tool_schema", map[string]any{"tool_name": "everything__ech"})
		require.True(t, result.IsError)

		var payload struct {
			Code    string `json:"code"`
			Details struct {
				Suggestions []string `json:"suggestions"`
			} `json:"details"`
		}
		require.NoError(t, json.Unmarshal([]byte(discoveryToolText(t, result)), &payload))
		assert.Equal(t, "tool_not_found", payload.Code)
		assert.Contains(t, payload.Details.Suggestions, "everything__echo")
	})

	t.Run("execute_tool forwards the call upstream", func(t *testing.T) {
		result := callDiscoveryTool(t, c, "execute_tool", map[string]any{
			"tool_name": "everything__echo",
			"arguments": map[string]any{"message": "hello from discovery e2e"},
		})
		require.False(t, result.IsError, "execute_tool failed: %s", discoveryToolText(t, result))
		assert.Contains(t, discoveryToolText(t, result), "hello from discovery e2e")
	})
}

func TestE2E_DiscoveryMode_EagerEndpointUnchanged(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	// The eager endpoint must keep exposing upstream tools directly.
	names := listMCPToolNames(t, newMCPProxyClient(t, env, ""))
	assert.Contains(t, names, "everything__echo")
	for _, metaTool := range discoveryMetaToolNames {
		assert.NotContains(t, names, metaTool, "eager endpoint must not expose meta-tools")
	}
}

func TestE2E_DiscoveryMode_LegacySSE(t *testing.T) {
	env := setupE2EServer(t, model.ModeDev)
	registerEverythingServer(t, env, "")

	c, err := client.NewSSEMCPClient(env.baseURL + "/sse?mode=discovery")
	require.NoError(t, err)
	require.NoError(t, c.Start(context.Background()))
	t.Cleanup(func() { _ = c.Close() })

	initResult, err := c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "e2e-discovery-sse", Version: "1.0.0"},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, initResult.Instructions, "on-demand tool discovery")
	assert.ElementsMatch(t, discoveryMetaToolNames, listMCPToolNames(t, c))

	result := callDiscoveryTool(t, c, "discover_tools", map[string]any{})
	require.False(t, result.IsError, "discover_tools failed: %s", discoveryToolText(t, result))

	var payload struct {
		Domains []struct {
			Name string `json:"name"`
		} `json:"domains"`
	}
	require.NoError(t, json.Unmarshal([]byte(discoveryToolText(t, result)), &payload))
	require.Len(t, payload.Domains, 1)
	assert.Equal(t, "everything", payload.Domains[0].Name)
}

func TestE2E_DiscoveryMode_EnterpriseAccessControl(t *testing.T) {
	env := setupE2EServer(t, model.ModeEnterprise)
	registerEverythingServer(t, env, env.adminToken)

	allowed, _ := newDiscoveryMCPClient(t, env, createMcpClient(t, env, "discovery-allowed", []string{"everything"}))
	denied, _ := newDiscoveryMCPClient(t, env, createMcpClient(t, env, "discovery-denied", []string{}))

	t.Run("allowed client sees the server", func(t *testing.T) {
		result := callDiscoveryTool(t, allowed, "discover_tools", map[string]any{})
		require.False(t, result.IsError, "discover_tools failed: %s", discoveryToolText(t, result))

		var payload struct {
			Domains []struct {
				Name string `json:"name"`
			} `json:"domains"`
		}
		require.NoError(t, json.Unmarshal([]byte(discoveryToolText(t, result)), &payload))
		require.Len(t, payload.Domains, 1)
		assert.Equal(t, "everything", payload.Domains[0].Name)
	})

	t.Run("denied client sees no servers", func(t *testing.T) {
		result := callDiscoveryTool(t, denied, "discover_tools", map[string]any{})
		require.False(t, result.IsError, "discover_tools failed: %s", discoveryToolText(t, result))

		var payload struct {
			Domains    []any `json:"domains"`
			TotalTools int64 `json:"total_tools"`
		}
		require.NoError(t, json.Unmarshal([]byte(discoveryToolText(t, result)), &payload))
		assert.Empty(t, payload.Domains)
		assert.Zero(t, payload.TotalTools)
	})

	t.Run("denied client cannot list an unauthorized server", func(t *testing.T) {
		result := callDiscoveryTool(t, denied, "discover_tools", map[string]any{"domain": "everything"})
		require.True(t, result.IsError)

		var payload struct {
			Code string `json:"code"`
		}
		require.NoError(t, json.Unmarshal([]byte(discoveryToolText(t, result)), &payload))
		assert.Equal(t, "domain_not_found", payload.Code)
	})

	t.Run("denied client cannot fetch schemas of unauthorized tools", func(t *testing.T) {
		result := callDiscoveryTool(t, denied, "get_tool_schema", map[string]any{"tool_name": "everything__echo"})
		require.True(t, result.IsError)

		var payload struct {
			Code string `json:"code"`
		}
		require.NoError(t, json.Unmarshal([]byte(discoveryToolText(t, result)), &payload))
		assert.Equal(t, "forbidden", payload.Code)
	})

	t.Run("denied client cannot execute unauthorized tools", func(t *testing.T) {
		result := callDiscoveryTool(t, denied, "execute_tool", map[string]any{
			"tool_name": "everything__echo",
			"arguments": map[string]any{"message": "should not run"},
		})
		require.True(t, result.IsError)

		var payload struct {
			Code string `json:"code"`
		}
		require.NoError(t, json.Unmarshal([]byte(discoveryToolText(t, result)), &payload))
		assert.Equal(t, "forbidden", payload.Code)
	})
}
