package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/version"
)

const (
	metaToolListTools     = "mcpjungle__list_tools"
	metaToolDescribeTools = "mcpjungle__describe_tools"
	metaToolCallTool      = "mcpjungle__call_tool"

	maxDescribeToolNames = 20

	maxLazyDescriptionLength = 140

	maxLazySuggestions = 10

	lazyMCPInstructions = "This MCP server exposes three meta-tools instead of the upstream tools themselves. " +
		"Use them in this order: " +
		"1. call mcpjungle__list_tools to see the canonical names and short descriptions of the tools you may use; " +
		"2. call mcpjungle__describe_tools with the canonical names you need to get their input schemas; " +
		"3. call mcpjungle__call_tool with a canonical tool name and its arguments to run it. " +
		"Never state that a capability is unavailable before calling mcpjungle__list_tools. " +
		"Never use a generic MCP context search as a substitute for this discovery flow."
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
		// The lazy tool set never changes, so the server does not advertise
		// support for tools/list_changed notifications.
		server.WithToolCapabilities(false),
		server.WithInstructions(lazyMCPInstructions),
	)
}

func (m *MCPService) initializeLazyHandlers() {
	m.lazyMcpProxyServer.AddTool(mcp.NewTool(
		metaToolListTools,
		mcp.WithDescription(
			"STEP 1. Call `mcpjungle__list_tools` to discover the tools you may use. "+
				"Returns one entry per available tool with its canonical `server__tool` name and a "+
				"short description. Never opens an upstream connection and never returns input schemas. "+
				"Do not guess tool names - only names returned here are valid.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("server",
			mcp.Description("Optional MCP server name. When set, only that server's tools are returned."),
		),
	), m.lazyListToolsHandler)

	m.lazyMcpProxyServer.AddTool(mcp.NewTool(
		metaToolDescribeTools,
		mcp.WithDescription(
			"STEP 2. Call `mcpjungle__describe_tools` with the canonical names returned by "+
				"`mcpjungle__list_tools` to get their descriptions and input schemas. "+
				"Request only the tools you are about to use. Never opens an upstream connection.",
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("names", mcp.Required(), mcp.WithStringItems(),
			mcp.Description("Canonical tool names returned by mcpjungle__list_tools. Between 1 and 20 names."),
		),
	), m.lazyDescribeToolsHandler)

	m.lazyMcpProxyServer.AddTool(mcp.NewTool(
		metaToolCallTool,
		mcp.WithDescription(
			"STEP 3. Call `mcpjungle__call_tool` to run a tool. This is the only way to invoke an "+
				"upstream tool in lazy mode. `name` must be a canonical `server__tool` name returned "+
				"by `mcpjungle__list_tools`. If you do not know the exact name, call "+
				"`mcpjungle__list_tools` first, and `mcpjungle__describe_tools` for the input schema.",
		),
		mcp.WithString("name", mcp.Required(),
			mcp.Description("Canonical tool name returned by mcpjungle__list_tools."),
		),
		mcp.WithAny("arguments",
			mcp.Description("Arguments for the tool as a JSON object matching its input schema. "+
				"Omit only for tools that take no arguments."),
		),
	), m.lazyCallToolHandler)
}

func (m *MCPService) lazyListToolsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tools, err := m.lazyVisibleTools(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	serverName := strings.TrimSpace(req.GetString("server", ""))
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		if serverName != "" {
			parent, _, ok := splitServerToolName(tool.Name)
			if !ok || parent != serverName {
				continue
			}
		}
		result = append(result, map[string]any{
			"name":        tool.Name,
			"description": truncateDescription(tool.Description, maxLazyDescriptionLength),
		})
	}
	return lazyJSONResult(result)
}

func (m *MCPService) lazyDescribeToolsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	names, err := lazyDescribeToolNames(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	allowed, err := m.lazyAllowedTools(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tools := make([]model.Tool, 0, len(names))
	for _, name := range names {
		tool, err := m.authorizeLazyTool(ctx, name, allowed)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tools = append(tools, *tool)
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })

	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		result = append(result, lazyToolMetadata(tool))
	}
	return lazyJSONResult(result)
}

// Authorization is re-evaluated on every call against the current registry
// state, so a tool that was disabled, removed from the tool group or denied to
// this client after discovery is never forwarded upstream.
func (m *MCPService) lazyCallToolHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := strings.TrimSpace(req.GetString("name", ""))
	if name == "" {
		return mcp.NewToolResultError(
			"`name` is required. Call mcpjungle__list_tools to discover the available tools.",
		), nil
	}

	allowed, err := m.lazyAllowedTools(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if _, err := m.authorizeLazyTool(ctx, name, allowed); err != nil {
		return mcp.NewToolResultError(m.lazyToolError(ctx, name, err, allowed)), nil
	}

	arguments, err := lazyToolArguments(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := m.MCPProxyToolCallHandler(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: name, Arguments: arguments},
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Describe and call share this current-state authorization check. The caller
// resolves the group set once per operation; list applies equivalent rules
// without per-tool database lookups.
func (m *MCPService) authorizeLazyTool(ctx context.Context, name string, allowed map[string]bool) (*model.Tool, error) {
	serverName, _, ok := splitServerToolName(name)
	if !ok {
		return nil, fmt.Errorf("%q is not a canonical tool name: %w", name, apierrors.ErrInvalidInput)
	}
	if err := authorizeProxyServerAccess(ctx, serverName); err != nil {
		return nil, err
	}

	serverModel, err := m.GetMcpServer(serverName)
	if err != nil {
		return nil, fmt.Errorf("MCP tool %s is unavailable: %w", name, err)
	}
	if !serverModel.Enabled {
		return nil, fmt.Errorf("MCP server %s is disabled", serverName)
	}

	tool, err := m.GetTool(name)
	if err != nil {
		return nil, fmt.Errorf("MCP tool %s is unavailable: %w", name, err)
	}
	if !tool.Enabled {
		return nil, fmt.Errorf("MCP tool %s is disabled", name)
	}

	if allowed != nil && !allowed[name] {
		return nil, fmt.Errorf("MCP tool %s is not available in this tool group", name)
	}
	return tool, nil
}

func (m *MCPService) lazyVisibleTools(ctx context.Context) ([]model.Tool, error) {
	allowed, err := m.lazyAllowedTools(ctx)
	if err != nil {
		return nil, err
	}
	return m.lazyVisibleToolsWithAllowed(ctx, allowed)
}

// lazyVisibleToolsWithAllowed applies the visibility policy with a group set
// that the caller has already resolved for this operation.
func (m *MCPService) lazyVisibleToolsWithAllowed(ctx context.Context, allowed map[string]bool) ([]model.Tool, error) {
	servers, err := m.ListEnabledMcpServers()
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP servers: %w", err)
	}
	enabledServers := make(map[string]bool, len(servers))
	for _, s := range servers {
		enabledServers[s.Name] = true
	}

	tools, err := m.ListTools()
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP tools: %w", err)
	}

	visible := make([]model.Tool, 0, len(tools))
	for _, tool := range tools {
		if !tool.Enabled || (allowed != nil && !allowed[tool.Name]) {
			continue
		}
		serverName, _, ok := splitServerToolName(tool.Name)
		if !ok || !enabledServers[serverName] {
			continue
		}
		if err := authorizeProxyServerAccess(ctx, serverName); err != nil {
			continue
		}
		visible = append(visible, tool)
	}
	sort.Slice(visible, func(i, j int) bool { return visible[i].Name < visible[j].Name })
	return visible, nil
}

// lazyAllowedTools resolves the effective tool set once for a group-scoped
// request. A nil set means there is no group filter; enterprise authorization
// still restricts which servers' tools surface. It never lists registry tools.
func (m *MCPService) lazyAllowedTools(ctx context.Context) (map[string]bool, error) {
	group := lazyGroupFromContext(ctx)
	if group == "" {
		return nil, nil
	}
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

// Unknown-name errors suggest only tools the caller is authorized to use.
func (m *MCPService) lazyToolError(ctx context.Context, name string, err error, allowed map[string]bool) string {
	if !errors.Is(err, apierrors.ErrNotFound) && !errors.Is(err, apierrors.ErrInvalidInput) {
		return err.Error()
	}
	message := fmt.Sprintf("MCP tool %q is not available. ", name)
	if suggestions := m.lazyToolSuggestions(ctx, name, allowed); len(suggestions) > 0 {
		message += fmt.Sprintf("Did you mean: %s. ", strings.Join(suggestions, ", "))
	}
	return message + "Call mcpjungle__list_tools to list the available tools."
}

// Suggestions come only from the visible set. Canonical names match tools on
// the same server; bare names match both server and tool components.
func (m *MCPService) lazyToolSuggestions(ctx context.Context, name string, allowed map[string]bool) []string {
	tools, err := m.lazyVisibleToolsWithAllowed(ctx, allowed)
	if err != nil {
		return nil
	}
	needle := strings.ToLower(name)
	serverPart, _, hasServer := strings.Cut(needle, "__")

	suggestions := make([]string, 0, maxLazySuggestions)
	for _, tool := range tools {
		toolServer, toolTool, _ := splitServerToolName(tool.Name)
		var matches bool
		if hasServer {
			matches = strings.ToLower(toolServer) == serverPart
		} else {
			matches = strings.Contains(strings.ToLower(toolTool), needle) ||
				strings.Contains(strings.ToLower(toolServer), needle)
		}
		if !matches {
			continue
		}
		suggestions = append(suggestions, tool.Name)
		if len(suggestions) == maxLazySuggestions {
			break
		}
	}
	return suggestions
}

// A single comma-separated string is accepted for clients that do not send
// an array.
func lazyDescribeToolNames(req mcp.CallToolRequest) ([]string, error) {
	var raw []string
	switch value := req.GetArguments()["names"].(type) {
	case nil:
		return nil, fmt.Errorf("`names` is required: pass an array of canonical tool names")
	case string:
		raw = strings.Split(value, ",")
	case []string:
		raw = value
	case []any:
		for _, item := range value {
			name, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("`names` must contain only strings")
			}
			raw = append(raw, name)
		}
	default:
		return nil, fmt.Errorf("`names` must be an array of canonical tool names")
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("`names` must contain at least one canonical tool name")
	}
	if len(raw) > maxDescribeToolNames {
		return nil, fmt.Errorf("`names` accepts at most %d tool names, got %d", maxDescribeToolNames, len(raw))
	}

	seen := make(map[string]bool, len(raw))
	names := make([]string, 0, len(raw))
	for _, name := range raw {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("`names` must not contain empty tool names")
		}
		if seen[name] {
			return nil, fmt.Errorf("`names` contains the duplicate tool name %q", name)
		}
		seen[name] = true
		names = append(names, name)
	}
	return names, nil
}

// lazyToolArguments extracts the arguments of a call_tool request.
//
// A JSON object is the expected form. A JSON object string is also accepted
// because some clients double-encode the payload.
func lazyToolArguments(req mcp.CallToolRequest) (map[string]any, error) {
	value, ok := req.GetArguments()["arguments"]
	if !ok || value == nil {
		return map[string]any{}, nil
	}
	switch typed := value.(type) {
	case map[string]any:
		return typed, nil
	case string:
		var parsed map[string]any
		if err := json.Unmarshal([]byte(typed), &parsed); err != nil {
			return nil, fmt.Errorf("`arguments` must be a JSON object: %w", err)
		}
		if parsed == nil {
			return map[string]any{}, nil
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("`arguments` must be a JSON object")
	}
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

// truncateDescription shortens a description so the list_tools response
// stays compact. The limit is counted in Unicode code points so multi-byte
// characters are never cut in half. Empty descriptions pass through unchanged.
func truncateDescription(desc string, max int) string {
	if desc == "" {
		return ""
	}
	runes := []rune(desc)
	if len(runes) <= max {
		return desc
	}
	return string(runes[:max-1]) + "…"
}
