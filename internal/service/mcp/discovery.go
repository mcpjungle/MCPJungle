package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/version"
)

const (
	// discoveryMcpServerName is the server name advertised to discovery-mode clients.
	discoveryMcpServerName = "MCPJungle Discovery Proxy"

	// Discovery meta-tool names. They match the gateway-style interface used by
	// external discovery gateways so clients can migrate without changing their workflow.
	metaToolDiscoverTools = "discover_tools"
	metaToolGetToolSchema = "get_tool_schema"
	metaToolExecuteTool   = "execute_tool"

	discoveryFormatSignatures = "signatures"
	discoveryFormatSchema     = "schema"

	// maxDiscoveryInstructionRunes caps the handshake instructions so a large
	// registry cannot make the initialize response unbounded.
	maxDiscoveryInstructionRunes = 4000

	// maxDiscoveryServerDescriptionRunes caps each server description in the
	// handshake overview.
	maxDiscoveryServerDescriptionRunes = 80

	// maxDiscoverySuggestions is the number of similar tool names reported when
	// a tool cannot be resolved.
	maxDiscoverySuggestions = 3

	// minDiscoverySuggestionSimilarity is the similarity ratio below which a
	// tool name is not worth suggesting.
	minDiscoverySuggestionSimilarity = 0.6
)

// Machine-readable error codes returned by the discovery meta-tools.
const (
	discoveryErrDomainNotFound   = "domain_not_found"
	discoveryErrToolNotFound     = "tool_not_found"
	discoveryErrForbidden        = "forbidden"
	discoveryErrInvalidArguments = "invalid_arguments"
	discoveryErrUpstreamError    = "upstream_error"
	discoveryErrInternal         = "internal_error"
)

const (
	discoveryInstructionsPreamble = "This is an MCP gateway with on-demand tool discovery. " +
		"It aggregates the MCP servers listed below. " +
		"Call `discover_tools(domain=\"<server>\")` to see that server's tools as signatures, " +
		"then `execute_tool` to run one."

	discoveryInstructionsFooter = "Workflow: discover_tools() → execute_tool(). " +
		"Call get_tool_schema() only when a signature is not enough. " +
		"Skip discovery for tools you've already used in this conversation."

	discoveryTruncationNotice = "… (server list truncated; call discover_tools() for the complete list)"
)

const (
	discoverToolsDescription = `Find the tools this gateway can run, and learn how to call them. ` +
		`Call it with no arguments to list the servers (each a set of related tools from one MCP server). ` +
		`Then pass domain to list one server's tools as Python-style signatures. ` +
		`With no arguments, it returns JSON: {"guidance", "domains": [{"name", "description", "tool_count"}], "total_tools"}. ` +
		`Otherwise it returns one signature per tool, followed by an indented line with the tool's description. ` +
		`Required parameters come first, and optional ones end in = None. ` +
		`Pass those names and values to execute_tool as arguments.`

	getToolSchemaDescription = `Get one tool's full JSON Schema for its arguments. ` +
		`Use it only when the signature from discover_tools is not enough. ` +
		`A signature shows each parameter's name, type, and top-level allowed values and bounds. ` +
		`It does not show what each parameter means, or which fields of a nested object are required. ` +
		`Returns JSON: {"name", "domain", "description", "parameters"}. ` +
		`The name must match exactly. ` +
		`An unknown name returns {"error", "code": "tool_not_found", "details": {"tool_name", "suggestions"}}.`

	executeToolDescription = `Run one tool and return its result. ` +
		`Call it once you know the tool's name and parameters, from discover_tools or from an earlier call in this conversation. ` +
		`The gateway sends the call to the server that owns the tool, so this can change data there. ` +
		`On failure the result carries a JSON error with a code field: ` +
		`tool_not_found means no tool has that name, upstream_error means the tool's server answered with an error.`
)

// ServerToolSummary is the per-server overview used by the discovery handshake
// instructions and by the no-argument discover_tools listing.
type ServerToolSummary struct {
	Name        string `json:"name" gorm:"column:name"`
	Description string `json:"description" gorm:"column:description"`
	ToolCount   int64  `json:"tool_count" gorm:"column:tool_count"`
}

// newDiscoveryMCPServer creates the MCP server that serves discovery-mode clients.
//
// Tool list-change notifications are disabled because the fixed meta-tool set never
// changes during the process lifetime. The handshake instructions carry the server
// overview so a model knows which groups exist without calling any tool.
func newDiscoveryMCPServer(instructions string) *server.MCPServer {
	return server.NewMCPServer(
		discoveryMcpServerName,
		version.GetVersion(),
		server.WithToolCapabilities(false),
		server.WithInstructions(instructions),
	)
}

// initDiscoveryTools registers the fixed discovery meta-tools on the discovery MCP server.
func (m *MCPService) initDiscoveryTools() {
	m.discoveryMcpProxyServer.AddTool(
		mcp.NewTool(
			metaToolDiscoverTools,
			mcp.WithDescription(discoverToolsDescription),
			mcp.WithString("domain",
				mcp.Description(
					`Server name from the no-argument listing. Lists that server's tools as signatures. `+
						`Omit it to list the servers.`,
				),
			),
			mcp.WithString("format",
				mcp.Description("Output format for a single-server listing."),
				mcp.Enum(discoveryFormatSignatures, discoveryFormatSchema),
			),
		),
		m.discoveryDiscoverToolsHandler,
	)
	m.discoveryMcpProxyServer.AddTool(
		mcp.NewTool(
			metaToolGetToolSchema,
			mcp.WithDescription(getToolSchemaDescription),
			mcp.WithString("tool_name",
				mcp.Required(),
				mcp.Description(`Canonical tool name in the form "server__tool".`),
			),
		),
		m.discoveryGetToolSchemaHandler,
	)
	m.discoveryMcpProxyServer.AddTool(
		mcp.NewTool(
			metaToolExecuteTool,
			mcp.WithDescription(executeToolDescription),
			mcp.WithString("tool_name",
				mcp.Required(),
				mcp.Description(`Canonical tool name in the form "server__tool".`),
			),
			mcp.WithObject("arguments",
				mcp.Required(),
				mcp.Description("Arguments for the tool, matching its input schema."),
				mcp.AdditionalProperties(true),
			),
		),
		m.discoveryExecuteToolHandler,
	)
}

// discoveryDiscoverToolsHandler lists either the server overview or one server's tool signatures.
func (m *MCPService) discoveryDiscoverToolsHandler(
	ctx context.Context, request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	domain := strings.TrimSpace(request.GetString("domain", ""))
	format := strings.TrimSpace(request.GetString("format", discoveryFormatSignatures))
	if format != discoveryFormatSchema {
		format = discoveryFormatSignatures
	}

	if domain == "" {
		return m.discoveryListDomainsResult(ctx), nil
	}
	return m.discoveryListServerToolsResult(ctx, domain, format), nil
}

// discoveryListDomainsResult returns the server overview visible to the caller.
func (m *MCPService) discoveryListDomainsResult(ctx context.Context) *mcp.CallToolResult {
	summaries, err := m.ListServerSummaries()
	if err != nil {
		return discoveryErrorResult(
			discoveryErrInternal,
			"failed to list MCP servers",
			map[string]any{"cause": err.Error()},
		)
	}

	visible := filterServerSummariesByAccess(ctx, summaries)
	var total int64
	for _, s := range visible {
		total += s.ToolCount
	}

	return discoveryJSONResult(map[string]any{
		"guidance":    discoveryInstructionsPreamble,
		"domains":     visible,
		"total_tools": total,
	})
}

// discoveryListServerToolsResult returns the tools of one server as signatures or JSON entries.
func (m *MCPService) discoveryListServerToolsResult(
	ctx context.Context, domain, format string,
) *mcp.CallToolResult {
	// Servers the caller cannot access are reported as not found, mirroring the
	// invisibility of unauthorized tools in the eager proxy's tools/list.
	srv, err := m.GetMcpServer(domain)
	if err != nil || !srv.Enabled || !clientHasServerAccess(ctx, domain) {
		return m.discoveryDomainNotFoundResult(ctx, domain)
	}

	tools, err := m.ListToolsByServer(domain)
	if err != nil {
		return discoveryErrorResult(
			discoveryErrInternal,
			"failed to list tools of MCP server "+domain,
			map[string]any{"cause": err.Error()},
		)
	}

	enabled := make([]model.Tool, 0, len(tools))
	for _, t := range tools {
		if t.Enabled {
			enabled = append(enabled, t)
		}
	}

	if format == discoveryFormatSchema {
		type toolEntry struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		entries := make([]toolEntry, 0, len(enabled))
		for _, t := range enabled {
			entries = append(entries, toolEntry{Name: t.Name, Description: collapseWhitespace(t.Description)})
		}
		return discoveryJSONResult(map[string]any{"domain": domain, "tools": entries})
	}

	var b strings.Builder
	for i, t := range enabled {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(renderToolSignature(t))
	}
	if b.Len() == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("MCP server %q exposes no enabled tools.", domain))
	}
	return mcp.NewToolResultText(b.String())
}

// discoveryGetToolSchemaHandler returns the full definition of a single tool.
func (m *MCPService) discoveryGetToolSchemaHandler(
	ctx context.Context, request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	name := strings.TrimSpace(request.GetString("tool_name", ""))
	if name == "" {
		return discoveryErrorResult(discoveryErrInvalidArguments, "tool_name is required", nil), nil
	}

	serverName, _, ok := splitServerToolName(name)
	if !ok {
		return m.discoveryToolNotFoundResult(ctx, name), nil
	}
	if err := authorizeProxyServerAccess(ctx, serverName); err != nil {
		return discoveryErrorResult(discoveryErrForbidden, err.Error(), nil), nil
	}

	tool, err := m.GetTool(name)
	if err != nil || !tool.Enabled {
		return m.discoveryToolNotFoundResult(ctx, name), nil
	}
	server, err := m.GetMcpServer(serverName)
	if err != nil || !server.Enabled {
		return m.discoveryToolNotFoundResult(ctx, name), nil
	}

	var parameters any = map[string]any{"type": "object"}
	if len(tool.InputSchema) > 0 && json.Valid(tool.InputSchema) {
		parameters = json.RawMessage(tool.InputSchema)
	}

	return discoveryJSONResult(map[string]any{
		"name":        tool.Name,
		"domain":      serverName,
		"description": collapseWhitespace(tool.Description),
		"parameters":  parameters,
	}), nil
}

// discoveryExecuteToolHandler forwards a tool call to the owning upstream server.
func (m *MCPService) discoveryExecuteToolHandler(
	ctx context.Context, request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	name := strings.TrimSpace(request.GetString("tool_name", ""))
	if name == "" {
		return discoveryErrorResult(discoveryErrInvalidArguments, "tool_name is required", nil), nil
	}

	serverName, _, ok := splitServerToolName(name)
	if !ok {
		return m.discoveryToolNotFoundResult(ctx, name), nil
	}
	if err := authorizeProxyServerAccess(ctx, serverName); err != nil {
		return discoveryErrorResult(discoveryErrForbidden, err.Error(), nil), nil
	}

	tool, err := m.GetTool(name)
	if err != nil || !tool.Enabled {
		return m.discoveryToolNotFoundResult(ctx, name), nil
	}
	server, err := m.GetMcpServer(serverName)
	if err != nil || !server.Enabled {
		return m.discoveryToolNotFoundResult(ctx, name), nil
	}

	args, _ := request.GetArguments()["arguments"].(map[string]any)
	if args == nil {
		args = map[string]any{}
	}

	proxyRequest := mcp.CallToolRequest{}
	proxyRequest.Params.Name = name
	proxyRequest.Params.Arguments = args

	result, err := m.MCPProxyToolCallHandler(ctx, proxyRequest)
	if err != nil {
		return discoveryErrorResult(discoveryErrUpstreamError, err.Error(), map[string]any{"tool": name}), nil
	}
	return result, nil
}

// discoveryDomainNotFoundResult reports an unknown or disabled server and lists
// the servers the caller can access.
func (m *MCPService) discoveryDomainNotFoundResult(
	ctx context.Context, domain string,
) *mcp.CallToolResult {
	available := []string{}
	if summaries, err := m.ListServerSummaries(); err == nil {
		for _, s := range filterServerSummariesByAccess(ctx, summaries) {
			available = append(available, s.Name)
		}
	}
	return discoveryErrorResult(
		discoveryErrDomainNotFound,
		fmt.Sprintf("MCP server %q is not available", domain),
		map[string]any{
			"domain":            domain,
			"available_domains": available,
		},
	)
}

// discoveryToolNotFoundResult reports an unknown or disabled tool with similar
// names the caller can try instead.
func (m *MCPService) discoveryToolNotFoundResult(
	ctx context.Context, name string,
) *mcp.CallToolResult {
	return discoveryErrorResult(
		discoveryErrToolNotFound,
		fmt.Sprintf("MCP tool %q is not available", name),
		map[string]any{
			"tool_name":   name,
			"suggestions": m.suggestToolNames(ctx, name),
		},
	)
}

// suggestToolNames returns up to maxDiscoverySuggestions tool names similar to
// the requested name, restricted to tools the caller can access.
func (m *MCPService) suggestToolNames(ctx context.Context, name string) []string {
	summaries, err := m.ListServerSummaries()
	if err != nil {
		return nil
	}
	enabledServers := make(map[string]bool, len(summaries))
	for _, s := range summaries {
		enabledServers[s.Name] = true
	}

	tools, err := m.ListTools()
	if err != nil {
		return nil
	}

	type candidate struct {
		name  string
		score float64
	}
	candidates := make([]candidate, 0, maxDiscoverySuggestions*4)
	lowered := strings.ToLower(name)
	for _, t := range tools {
		if !t.Enabled {
			continue
		}
		serverName, _, ok := splitServerToolName(t.Name)
		if !ok || !enabledServers[serverName] || !clientHasServerAccess(ctx, serverName) {
			continue
		}
		score := similarityRatio(lowered, strings.ToLower(t.Name))
		if score < minDiscoverySuggestionSimilarity {
			continue
		}
		candidates = append(candidates, candidate{name: t.Name, score: score})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].name < candidates[j].name
	})

	limit := min(len(candidates), maxDiscoverySuggestions)
	suggestions := make([]string, 0, limit)
	for _, c := range candidates[:limit] {
		suggestions = append(suggestions, c.name)
	}
	return suggestions
}

// filterServerSummariesByAccess narrows the server overview to the servers the
// caller may access in enterprise mode. It fails closed when the caller context
// is missing, because that means the request bypassed the auth middleware.
func filterServerSummariesByAccess(
	ctx context.Context, summaries []ServerToolSummary,
) []ServerToolSummary {
	mode, ok := ctx.Value("mode").(model.ServerMode)
	if !ok {
		return nil
	}
	if !model.IsEnterpriseMode(mode) {
		return summaries
	}
	client, ok := ctx.Value("client").(*model.McpClient)
	if !ok || client == nil {
		return nil
	}
	visible := make([]ServerToolSummary, 0, len(summaries))
	for _, s := range summaries {
		if client.CheckHasServerAccess(s.Name) {
			visible = append(visible, s)
		}
	}
	return visible
}

func clientHasServerAccess(ctx context.Context, serverName string) bool {
	mode, ok := ctx.Value("mode").(model.ServerMode)
	if !ok {
		return false
	}
	if !model.IsEnterpriseMode(mode) {
		return true
	}
	client, ok := ctx.Value("client").(*model.McpClient)
	if !ok || client == nil {
		return false
	}
	return client.CheckHasServerAccess(serverName)
}

// buildDiscoveryInstructions renders the discovery handshake instructions.
// The server overview is what lets a model learn which groups exist without
// calling any tool, so it must stay complete unless the size cap is hit.
func buildDiscoveryInstructions(summaries []ServerToolSummary) string {
	var b strings.Builder
	b.WriteString(discoveryInstructionsPreamble)
	b.WriteString("\n\nServers:\n")

	if len(summaries) == 0 {
		b.WriteString("(no servers are registered)\n")
	}

	for _, s := range summaries {
		line := fmt.Sprintf("- **%s** (%d tools)", s.Name, s.ToolCount)
		if desc := truncateRunes(collapseWhitespace(s.Description), maxDiscoveryServerDescriptionRunes); desc != "" {
			line += " — " + desc
		}
		line += "\n"

		// Reserve room for the footer and the truncation notice so the
		// instructions never exceed the cap.
		reserve := runeCount(discoveryInstructionsFooter) + runeCount(discoveryTruncationNotice) + 3
		if runeCount(b.String())+runeCount(line)+reserve > maxDiscoveryInstructionRunes {
			b.WriteString(discoveryTruncationNotice)
			b.WriteString("\n")
			break
		}
		b.WriteString(line)
	}

	b.WriteString("\n")
	b.WriteString(discoveryInstructionsFooter)
	return b.String()
}

// renderToolSignature renders one tool as a Python-style signature plus its description.
func renderToolSignature(t model.Tool) string {
	var schema mcp.ToolInputSchema
	if len(t.InputSchema) > 0 {
		if err := json.Unmarshal(t.InputSchema, &schema); err != nil {
			schema = mcp.ToolInputSchema{}
		}
	}

	signature := fmt.Sprintf("%s(%s) -> any", t.Name, renderToolParams(schema))
	if desc := collapseWhitespace(t.Description); desc != "" {
		signature += "\n  " + desc
	}
	return signature
}

// renderToolParams renders the parameter list of a tool signature. Required
// parameters keep the schema's declared order; optional ones are sorted by name
// so repeated listings are stable.
func renderToolParams(schema mcp.ToolInputSchema) string {
	required := make(map[string]bool, len(schema.Required))
	for _, name := range schema.Required {
		required[name] = true
	}

	parts := make([]string, 0, len(schema.Properties))
	for _, name := range schema.Required {
		prop, _ := schema.Properties[name].(map[string]any)
		parts = append(parts, renderRequiredToolParam(name, prop))
	}

	optional := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		if !required[name] {
			optional = append(optional, name)
		}
	}
	sort.Strings(optional)
	for _, name := range optional {
		prop, _ := schema.Properties[name].(map[string]any)
		parts = append(parts, renderOptionalToolParam(name, prop))
	}

	return strings.Join(parts, ", ")
}

func renderRequiredToolParam(name string, prop map[string]any) string {
	return name + ": " + formatToolParamType(name, prop)
}

func renderOptionalToolParam(name string, prop map[string]any) string {
	return name + ": " + formatToolParamType(name, prop) + " = None"
}

func formatToolParamType(name string, prop map[string]any) string {
	typeName := formatSchemaType(prop, 0)
	if constraints := formatParamConstraints(name, prop); len(constraints) > 0 {
		quoted := make([]string, 0, len(constraints))
		for _, c := range constraints {
			quoted = append(quoted, "'"+c+"'")
		}
		typeName = fmt.Sprintf("Annotated[%s, %s]", typeName, strings.Join(quoted, ", "))
	}
	return typeName
}

func formatParamConstraints(name string, prop map[string]any) []string {
	if prop == nil {
		return nil
	}
	var out []string
	if v, ok := numberValue(prop["minimum"]); ok {
		out = append(out, fmt.Sprintf("%s <= %s", formatNumber(v), name))
	}
	if v, ok := numberValue(prop["maximum"]); ok {
		out = append(out, fmt.Sprintf("%s <= %s", name, formatNumber(v)))
	}
	if v, ok := numberValue(prop["exclusiveMinimum"]); ok {
		out = append(out, fmt.Sprintf("%s < %s", formatNumber(v), name))
	}
	if v, ok := numberValue(prop["exclusiveMaximum"]); ok {
		out = append(out, fmt.Sprintf("%s < %s", name, formatNumber(v)))
	}
	if v, ok := numberValue(prop["minLength"]); ok {
		out = append(out, fmt.Sprintf("len(%s) >= %s", name, formatNumber(v)))
	}
	if v, ok := numberValue(prop["maxLength"]); ok {
		out = append(out, fmt.Sprintf("len(%s) <= %s", name, formatNumber(v)))
	}
	return out
}

// formatSchemaType renders a JSON schema fragment as a Python-style type
// annotation. depth tracks how many object levels have been inlined; objects
// nested deeper than one level render as dict so signatures stay readable.
func formatSchemaType(prop map[string]any, depth int) string {
	if prop == nil {
		return "any"
	}

	if values, ok := prop["enum"].([]any); ok && len(values) > 0 {
		quoted := make([]string, 0, len(values))
		for _, v := range values {
			quoted = append(quoted, strconv.Quote(fmt.Sprint(v)))
		}
		return "Literal[" + strings.Join(quoted, ", ") + "]"
	}

	if alternatives := unionAlternatives(prop); len(alternatives) > 0 {
		parts := make([]string, 0, len(alternatives))
		for _, alt := range alternatives {
			altMap, _ := alt.(map[string]any)
			parts = append(parts, formatSchemaType(altMap, depth))
		}
		return strings.Join(parts, " | ")
	}

	switch t := prop["type"].(type) {
	case string:
		return formatSchemaTypeName(t, prop, depth)
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			if name, ok := item.(string); ok {
				parts = append(parts, formatSchemaTypeName(name, prop, depth))
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " | ")
		}
	}

	// A schema without an explicit type but with properties describes an object.
	if _, ok := prop["properties"]; ok {
		return formatSchemaTypeName("object", prop, depth)
	}
	return "any"
}

func formatSchemaTypeName(typeName string, prop map[string]any, depth int) string {
	switch typeName {
	case "string":
		return "str"
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "null":
		return "None"
	case "array":
		items, _ := prop["items"].(map[string]any)
		return "list[" + formatSchemaType(items, depth+1) + "]"
	case "object":
		if depth >= 1 {
			return "dict"
		}
		properties, _ := prop["properties"].(map[string]any)
		if len(properties) == 0 {
			return "dict"
		}
		keys := make([]string, 0, len(properties))
		for k := range properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, 0, len(keys))
		for _, k := range keys {
			child, _ := properties[k].(map[string]any)
			pairs = append(pairs, strconv.Quote(k)+": "+formatSchemaType(child, depth+1))
		}
		return "{" + strings.Join(pairs, ", ") + "}"
	}
	return "any"
}

func unionAlternatives(prop map[string]any) []any {
	for _, key := range []string{"anyOf", "oneOf"} {
		if alternatives, ok := prop[key].([]any); ok && len(alternatives) > 0 {
			return alternatives
		}
	}
	return nil
}

func numberValue(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func formatNumber(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func runeCount(s string) int {
	return len([]rune(s))
}

func truncateRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "…"
}

// similarityRatio returns a 0..1 similarity between two strings based on the
// Levenshtein distance, mirroring difflib's ratio.
func similarityRatio(a, b string) float64 {
	if a == b {
		return 1
	}
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 1
	}
	return 1 - float64(levenshteinDistance(a, b))/float64(maxLen)
}

func levenshteinDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

func discoveryJSONResult(payload any) *mcp.CallToolResult {
	data, err := json.Marshal(payload)
	if err != nil {
		return discoveryErrorResult(
			discoveryErrInternal,
			"failed to encode result",
			map[string]any{"cause": err.Error()},
		)
	}
	return mcp.NewToolResultText(string(data))
}

func discoveryErrorResult(code, message string, details map[string]any) *mcp.CallToolResult {
	payload := map[string]any{"error": message, "code": code}
	if len(details) > 0 {
		payload["details"] = details
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("%s: %s", code, message))
	}
	return mcp.NewToolResultError(string(data))
}
