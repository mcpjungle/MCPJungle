package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/version"
)

const (
	metaToolListServers     = "mcpjungle__list_servers"
	metaToolShowServerTools = "mcpjungle__show_server_tools"
	metaToolInvoke          = "mcpjungle__invoke"

	lazyMCPInstructions = "This is a lazy MCP gateway. Before saying an MCP server or tool is unavailable, call mcpjungle__list_servers to discover the registered services. When a task needs one of those services, call mcpjungle__show_server_tools with its exact server name. That activates the selected service's canonical tools in this session. Call an activated tool directly after tools/list_changed. If the client does not dynamically expose the activated tools, call mcpjungle__invoke with the canonical tool name and JSON arguments instead. Never use a generic MCP context search as a substitute for this discovery flow."
)

type lazyContextKey string

const lazyGroupContextKey lazyContextKey = "lazy_group"

// ToolGroupResolver is the subset of ToolGroupService used by lazy mode.
type ToolGroupResolver interface {
	ResolveEffectiveTools(name string) ([]string, error)
}

// WithLazyGroup binds a lazy request to an optional tool group.
func (m *MCPService) WithLazyGroup(ctx context.Context, group string) context.Context {
	return context.WithValue(ctx, lazyGroupContextKey, group)
}

func lazyGroupFromContext(ctx context.Context) string {
	group, _ := ctx.Value(lazyGroupContextKey).(string)
	return group
}

func newLazyMCPServer() *server.MCPServer {
	return server.NewMCPServer(
		"MCPJungle lazy MCP proxy",
		version.GetVersion(),
		server.WithToolCapabilities(true),
		server.WithInstructions(lazyMCPInstructions),
	)
}

// initializeLazyHandlers registers the fixed lazy tool set once at startup.
//
// The lazy proxy exposes exactly two meta-tools so an agent has a clear,
// two-step discovery workflow:
//
//  1. Call `mcpjungle__list_servers` to discover every enabled MCP server the
//     caller is allowed to see (no upstream connection is opened). The
//     response includes each server's description and the count of enabled
//     tools it provides, so the agent can decide whether to drill in.
//  2. Call `mcpjungle__show_server_tools` with the chosen `server` to fetch
//     and activate the full tool list (name, description, input_schema) for
//     that one server. The canonical `server__tool` names are added only to
//     the current MCP session and can then be invoked directly.
//
// Neither meta-tool opens an upstream session. Activating a server changes only
// the current client session and is safe to repeat.
func (m *MCPService) initializeLazyHandlers() {
	m.lazyMcpProxyServer.AddTool(mcp.NewTool(
		metaToolListServers,
		mcp.WithDescription(
			"FIRST STEP. Call `mcpjungle__list_servers` to discover the enabled MCP servers "+
				"available through MCPJungle. Returns one entry per enabled server with the server "+
				"name, a short description, the transport, and the count of enabled tools. "+
				"Never opens an upstream connection. Pick a server from the response, then call "+
				"`mcpjungle__show_server_tools` with `server=<name>` to fetch its tools. "+
				"Do not guess server names \u2014 only names returned here are valid.",
		),
	), m.lazyListServersHandler)
	m.lazyMcpProxyServer.AddTool(mcp.NewTool(
		metaToolShowServerTools,
		mcp.WithDescription(
			"SECOND STEP. Call `mcpjungle__show_server_tools` after `mcpjungle__list_servers` "+
				"to activate the tools for a single MCP server in this MCP session. The `server` argument "+
				"is REQUIRED and must be a server name copied verbatim from a `mcpjungle__list_servers` "+
				"result. It returns each tool's canonical `server__tool` name, description, and input schema, "+
				"then sends a tools/list_changed notification so those canonical tools can be invoked directly. "+
				"Do NOT call this tool without `server` — it is intentionally rejected to avoid "+
				"dumping every registered tool into context.",
		),
		mcp.WithString("server", mcp.Required(),
			mcp.Description("MCP server name copied verbatim from a `mcpjungle__list_servers` result. Required."),
		),
	), m.lazyShowServerToolsHandler)
	m.lazyMcpProxyServer.AddTool(mcp.NewTool(
		metaToolInvoke,
		mcp.WithDescription(
			"FALLBACK ONLY. Invoke an enabled, authorized canonical `server__tool` when the MCP client does not dynamically expose tools activated by mcpjungle__show_server_tools. Call show_server_tools first to obtain the exact tool name and schema. Pass `arguments` as a JSON object string matching that schema.",
		),
		mcp.WithString("name", mcp.Required(), mcp.Description("Canonical tool name returned by mcpjungle__show_server_tools.")),
		mcp.WithString("arguments", mcp.Required(), mcp.Description("JSON object string matching the selected tool's input schema.")),
	), m.lazyInvokeHandler)
}

// lazyListServersHandler returns the enabled MCP servers the caller can see.
//
// It applies the same enterprise allow-list enforcement as the eager proxy
// (via `authorizeProxyServerAccess`) and, for tool-group requests, hides
// servers whose tools are not exposed by the group. Disabled servers are
// filtered at the SQL level by `ListEnabledMcpServers`. The response never
// opens an upstream connection.
func (m *MCPService) lazyListServersHandler(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	allowed, err := m.lazyAllowedTools(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	servers, err := m.ListEnabledMcpServers()
	if err != nil {
		return mcp.NewToolResultErrorf("failed to list MCP servers: %v", err), nil
	}

	// Pre-compute the enabled tool count per server in one query so this
	// handler stays O(N servers) regardless of how many tools exist.
	type toolCountRow struct {
		ServerID uint `gorm:"column:server_id"`
		Count    int  `gorm:"column:c"`
	}
	var counts []toolCountRow
	if err := m.db.Model(&model.Tool{}).
		Select("server_id, COUNT(*) AS c").
		Where("enabled = ?", true).
		Group("server_id").
		Scan(&counts).Error; err != nil {
		return mcp.NewToolResultErrorf("failed to count tools: %v", err), nil
	}
	countByServer := make(map[uint]int, len(counts))
	for _, row := range counts {
		countByServer[row.ServerID] = row.Count
	}

	result := make([]map[string]any, 0, len(servers))
	for _, s := range servers {
		if err := authorizeProxyServerAccess(ctx, s.Name); err != nil {
			continue
		}
		// For tool-group requests, hide servers that contribute no tools
		// to the group (mirrors the filter applied by `authorizeLazyServer`).
		if lazyGroupFromContext(ctx) != "" {
			hasGroupTool := false
			for toolName := range allowed {
				parent, _, _ := splitServerToolName(toolName)
				if parent == s.Name {
					hasGroupTool = true
					break
				}
			}
			if !hasGroupTool {
				continue
			}
		}

		entry := map[string]any{
			"name":               s.Name,
			"description":        truncateDescription(s.Description, 140),
			"transport":          string(s.Transport),
			"enabled_tool_count": countByServer[s.ID],
		}
		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool { return result[i]["name"].(string) < result[j]["name"].(string) })
	return lazyJSONResult(result)
}

// lazyShowServerToolsHandler activates the enabled tools for one MCP server in
// the current session and returns their metadata as confirmation.
//
// `server` is required. Without it we return an explicit error pointing the
// agent at `mcpjungle__list_servers`, instead of dumping the global tool
// list. The handler authorizes the server through `authorizeLazyServer` and
// applies `lazyAllowedTools` for tool-group requests.
func (m *MCPService) lazyShowServerToolsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clientSession := server.ClientSessionFromContext(ctx)
	if clientSession == nil {
		return mcp.NewToolResultError("lazy tool activation requires an active MCP session"), nil
	}

	serverName := strings.TrimSpace(req.GetString("server", ""))
	if serverName == "" {
		return mcp.NewToolResultError(
			"`server` is required. Call mcpjungle__list_servers first to discover the available servers.",
		), nil
	}

	if err := m.authorizeLazyServer(ctx, serverName); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tools, err := m.ListToolsByServer(serverName)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to list tools for %s: %v", serverName, err), nil
	}

	allowed, err := m.lazyAllowedTools(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := make([]map[string]any, 0, len(tools))
	sessionTools := make([]server.ServerTool, 0, len(tools))
	for _, tool := range tools {
		if !tool.Enabled || !allowed[tool.Name] {
			continue
		}
		mcpTool, err := convertToolModelToMcpObject(&tool)
		if err != nil {
			return mcp.NewToolResultErrorf("failed to prepare tool %s: %v", tool.Name, err), nil
		}
		sessionTools = append(sessionTools, server.ServerTool{
			Tool:    mcpTool,
			Handler: m.MCPProxyToolCallHandler,
		})
		result = append(result, lazyToolMetadata(tool))
	}
	if err := m.lazyMcpProxyServer.AddSessionTools(clientSession.SessionID(), sessionTools...); err != nil {
		return mcp.NewToolResultErrorf("failed to activate tools for %s: %v", serverName, err), nil
	}
	sort.Slice(result, func(i, j int) bool { return result[i]["name"].(string) < result[j]["name"].(string) })
	return lazyJSONResult(result)
}

// lazyInvokeHandler is a compatibility fallback for MCP clients that receive
// tools/list_changed but do not add the newly activated tools to their callable
// registry. It enforces the same enabled-tool and server authorization checks
// as direct proxy calls before forwarding the request upstream.
func (m *MCPService) lazyInvokeHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := strings.TrimSpace(req.GetString("name", ""))
	serverName, _, ok := splitServerToolName(name)
	if !ok {
		return mcp.NewToolResultError("`name` must be a canonical `server__tool` name returned by mcpjungle__show_server_tools"), nil
	}
	if err := m.authorizeLazyServer(ctx, serverName); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tool, err := m.GetTool(name)
	if err != nil || !tool.Enabled {
		return mcp.NewToolResultErrorf("MCP tool %s is unavailable", name), nil
	}

	var arguments map[string]any
	if err := json.Unmarshal([]byte(req.GetString("arguments", "")), &arguments); err != nil {
		return mcp.NewToolResultErrorf("`arguments` must be a JSON object: %v", err), nil
	}
	result, err := m.MCPProxyToolCallHandler(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Name: name, Arguments: arguments}})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// lazyAllowedTools returns the set of enabled tool canonical names the
// caller is allowed to invoke. In enterprise mode without a tool group it
// still returns every enabled tool — `authorizeProxyServerAccess` is the
// boundary that restricts which servers' tools actually surface.
func (m *MCPService) lazyAllowedTools(ctx context.Context) (map[string]bool, error) {
	if group := lazyGroupFromContext(ctx); group != "" {
		if m.toolGroupService == nil {
			return nil, fmt.Errorf("tool groups are unavailable")
		}
		tools, err := m.toolGroupService.ResolveEffectiveTools(group)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve tool group %q: %w", group, err)
		}
		allowed := make(map[string]bool, len(tools))
		for _, tool := range tools {
			allowed[tool] = true
		}
		return allowed, nil
	}
	tools, err := m.ListTools()
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP tools: %w", err)
	}
	allowed := make(map[string]bool, len(tools))
	for _, tool := range tools {
		if tool.Enabled {
			allowed[tool.Name] = true
		}
	}
	return allowed, nil
}

// authorizeLazyServer confirms a server exists, is enabled, is accessible to
// the caller under the enterprise allow list, and (for tool-group requests)
// contributes at least one allowed tool. Used by `lazyShowServerToolsHandler`
// and `lazyListServersHandler`.
func (m *MCPService) authorizeLazyServer(ctx context.Context, serverName string) error {
	serverModel, err := m.GetMcpServer(serverName)
	if err != nil || !serverModel.Enabled {
		return fmt.Errorf("MCP server %s is unavailable", serverName)
	}
	if err := authorizeProxyServerAccess(ctx, serverName); err != nil {
		return err
	}
	allowed, err := m.lazyAllowedTools(ctx)
	if err != nil {
		return err
	}
	for toolName := range allowed {
		parent, _, _ := splitServerToolName(toolName)
		if parent == serverName {
			return nil
		}
	}
	return fmt.Errorf("MCP server %s is unavailable in this tool group", serverName)
}

func lazyToolMetadata(tool model.Tool) map[string]any {
	var schema any = map[string]any{}
	if len(tool.InputSchema) > 0 {
		_ = json.Unmarshal(tool.InputSchema, &schema)
	}
	return map[string]any{"name": tool.Name, "description": tool.Description, "input_schema": schema}
}

func lazyJSONResult(value any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(data)), nil
}

// truncateDescription shortens a server description so the list_servers
// response stays compact. Empty descriptions pass through unchanged.
func truncateDescription(desc string, max int) string {
	if desc == "" {
		return ""
	}
	if len(desc) <= max {
		return desc
	}
	return desc[:max-1] + "\u2026"
}
