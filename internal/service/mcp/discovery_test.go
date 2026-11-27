package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/telemetry"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDiscoveryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(&model.McpServer{}, &model.Tool{}); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}
	return db
}

func newDiscoveryTestServer(t *testing.T, db *gorm.DB, name, description string) *model.McpServer {
	t.Helper()

	server, err := model.NewStdioServer(name, description, "test-cmd", nil, nil, "")
	if err != nil {
		t.Fatalf("failed to build test server %s: %v", name, err)
	}
	if err := db.Create(server).Error; err != nil {
		t.Fatalf("failed to create test server %s: %v", name, err)
	}
	return server
}

// newDiscoveryTestTool creates a tool with the requested enabled state.
// The Enabled column has a DB default of true, so disabling takes a second
// update; creating with Enabled=false would silently store true.
func newDiscoveryTestTool(
	t *testing.T, db *gorm.DB, server *model.McpServer, name, description, schema string, enabled bool,
) {
	t.Helper()

	tool := &model.Tool{
		ServerID:    server.ID,
		Name:        name,
		Description: description,
		InputSchema: datatypes.JSON(schema),
	}
	if err := db.Create(tool).Error; err != nil {
		t.Fatalf("failed to create test tool %s: %v", name, err)
	}
	if !enabled {
		if err := db.Model(tool).Update("enabled", false).Error; err != nil {
			t.Fatalf("failed to disable test tool %s: %v", name, err)
		}
	}
}

// seedDiscoveryFixture registers two enabled servers, one disabled server and a
// mix of enabled and disabled tools.
func seedDiscoveryFixture(t *testing.T, db *gorm.DB) {
	t.Helper()

	jira := newDiscoveryTestServer(t, db, "jira", "Issue tracker")
	wiki := newDiscoveryTestServer(t, db, "wiki", "")
	old := newDiscoveryTestServer(t, db, "old", "Disabled server")
	if err := db.Model(old).Update("enabled", false).Error; err != nil {
		t.Fatalf("failed to disable test server: %v", err)
	}

	jiraSchema := `{"type":"object","properties":{"query":{"type":"string"},` +
		`"limit":{"type":"integer","minimum":1,"maximum":100}},"required":["query"]}`
	newDiscoveryTestTool(t, db, jira, "search", "Search issues in a project", jiraSchema, true)
	newDiscoveryTestTool(t, db, jira, "create_issue", "Create a new issue", `{"type":"object","properties":{}}`, true)
	newDiscoveryTestTool(t, db, jira, "archived_tool", "Disabled tool", `{"type":"object","properties":{}}`, false)

	wikiSchema := `{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`
	newDiscoveryTestTool(t, db, wiki, "confluence_search", "Search Confluence pages", wikiSchema, true)

	newDiscoveryTestTool(t, db, old, "legacy", "Tool on a disabled server", `{"type":"object","properties":{}}`, true)
}

func newDiscoveryTestService(t *testing.T, db *gorm.DB) *MCPService {
	t.Helper()
	return &MCPService{db: db, metrics: telemetry.NewNoopCustomMetrics()}
}

func devModeContext() context.Context {
	return context.WithValue(context.Background(), "mode", model.ModeDev)
}

func enterpriseClientContext(client *model.McpClient) context.Context {
	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	return context.WithValue(ctx, "client", client)
}

func newDiscoveryCallRequest(args map[string]any) mcp.CallToolRequest {
	request := mcp.CallToolRequest{}
	request.Params.Arguments = args
	return request
}

func discoveryResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil {
		t.Fatalf("result is nil")
	}
	if len(result.Content) == 0 {
		t.Fatalf("result has no content")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("result content is %T, want mcp.TextContent", result.Content[0])
	}
	return text.Text
}

func TestListServerSummaries(t *testing.T) {
	db := setupDiscoveryTestDB(t)
	seedDiscoveryFixture(t, db)
	service := newDiscoveryTestService(t, db)

	summaries, err := service.ListServerSummaries()
	if err != nil {
		t.Fatalf("ListServerSummaries() error = %v", err)
	}

	if len(summaries) != 2 {
		t.Fatalf("got %d summaries, want 2 (disabled server must be excluded): %+v", len(summaries), summaries)
	}
	if summaries[0].Name != "jira" || summaries[0].ToolCount != 2 || summaries[0].Description != "Issue tracker" {
		t.Fatalf("unexpected jira summary: %+v", summaries[0])
	}
	if summaries[1].Name != "wiki" || summaries[1].ToolCount != 1 || summaries[1].Description != "" {
		t.Fatalf("unexpected wiki summary: %+v", summaries[1])
	}
}

func TestBuildDiscoveryInstructions(t *testing.T) {
	t.Run("empty registry", func(t *testing.T) {
		got := buildDiscoveryInstructions(nil)
		if !strings.Contains(got, "(no servers are registered)") {
			t.Fatalf("instructions do not mention an empty registry: %q", got)
		}
	})

	t.Run("overview keeps order, counts and descriptions", func(t *testing.T) {
		got := buildDiscoveryInstructions([]ServerToolSummary{
			{Name: "jira", Description: "Issue tracker", ToolCount: 61},
			{Name: "wiki", ToolCount: 35},
		})
		for _, want := range []string{
			"- **jira** (61 tools) — Issue tracker",
			"- **wiki** (35 tools)",
			discoveryInstructionsPreamble,
			discoveryInstructionsFooter,
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("instructions missing %q:\n%s", want, got)
			}
		}
		if strings.Contains(got, "- **wiki** (35 tools) —") {
			t.Fatalf("empty description must not add a separator:\n%s", got)
		}
	})

	t.Run("long description is truncated", func(t *testing.T) {
		long := strings.Repeat("长", 100)
		got := buildDiscoveryInstructions([]ServerToolSummary{
			{Name: "kubemcp", Description: long, ToolCount: 7},
		})
		if !strings.Contains(got, strings.Repeat("长", maxDiscoveryServerDescriptionRunes)+"…") {
			t.Fatalf("description was not truncated to %d runes:\n%s", maxDiscoveryServerDescriptionRunes, got)
		}
	})

	t.Run("descriptions collapse newlines", func(t *testing.T) {
		got := buildDiscoveryInstructions([]ServerToolSummary{
			{Name: "svc", Description: "first line\nsecond  line", ToolCount: 1},
		})
		if !strings.Contains(got, "— first line second line") {
			t.Fatalf("description newlines not collapsed:\n%s", got)
		}
	})

	t.Run("size cap truncates the listing", func(t *testing.T) {
		summaries := make([]ServerToolSummary, 0, 300)
		for i := 0; i < 300; i++ {
			summaries = append(summaries, ServerToolSummary{
				Name:        "server-" + strings.Repeat("x", 10),
				Description: strings.Repeat("d", 60),
				ToolCount:   10,
			})
		}
		got := buildDiscoveryInstructions(summaries)
		if runeCount(got) > maxDiscoveryInstructionRunes {
			t.Fatalf("instructions exceed the cap: %d runes", runeCount(got))
		}
		if !strings.Contains(got, discoveryTruncationNotice) {
			t.Fatalf("instructions do not carry the truncation notice")
		}
	})
}

func TestRenderToolSignature(t *testing.T) {
	tests := []struct {
		name string
		tool model.Tool
		want string
	}{
		{
			name: "no parameters",
			tool: model.Tool{Name: "jira__noop"},
			want: "jira__noop() -> any",
		},
		{
			name: "required first, optional sorted",
			tool: model.Tool{
				Name:        "jira__search",
				Description: "Search issues",
				InputSchema: datatypes.JSON(
					`{"type":"object","properties":{"query":{"type":"string"},"limit":{"type":"integer"}},` +
						`"required":["query"]}`,
				),
			},
			want: "jira__search(query: str, limit: int = None) -> any\n  Search issues",
		},
		{
			name: "bounds render as Annotated",
			tool: model.Tool{
				Name: "jira__search",
				InputSchema: datatypes.JSON(
					`{"type":"object","properties":{"limit":{"type":"integer","minimum":1,"maximum":100}}}`,
				),
			},
			want: "jira__search(limit: Annotated[int, '1 <= limit', 'limit <= 100'] = None) -> any",
		},
		{
			name: "enum renders as Literal",
			tool: model.Tool{
				Name: "jira__transition",
				InputSchema: datatypes.JSON(
					`{"type":"object","properties":{"status":{"type":"string","enum":["open","closed"]}}}`,
				),
			},
			want: `jira__transition(status: Literal["open", "closed"] = None) -> any`,
		},
		{
			name: "nested objects inline one level then collapse to dict",
			tool: model.Tool{
				Name: "jira__query",
				InputSchema: datatypes.JSON(
					`{"type":"object","properties":{"options":{"type":"object","properties":{` +
						`"a":{"type":"string"},"b":{"type":"object","properties":{"deep":{"type":"string"}}}}}}}`,
				),
			},
			want: `jira__query(options: {"a": str, "b": dict} = None) -> any`,
		},
		{
			name: "array of strings",
			tool: model.Tool{
				Name: "jira__batch",
				InputSchema: datatypes.JSON(
					`{"type":"object","properties":{"tags":{"type":"array","items":{"type":"string"}}}}`,
				),
			},
			want: "jira__batch(tags: list[str] = None) -> any",
		},
		{
			name: "nullable union type",
			tool: model.Tool{
				Name: "jira__assign",
				InputSchema: datatypes.JSON(
					`{"type":"object","properties":{"user":{"type":["string","null"]}}}`,
				),
			},
			want: "jira__assign(user: str | None = None) -> any",
		},
		{
			name: "description newlines collapse",
			tool: model.Tool{
				Name:        "jira__create",
				Description: "Create an\nissue",
				InputSchema: datatypes.JSON(`{"type":"object","properties":{}}`),
			},
			want: "jira__create() -> any\n  Create an issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderToolSignature(tt.tool); got != tt.want {
				t.Fatalf("renderToolSignature() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestDiscoveryDiscoverTools_ListsDomains(t *testing.T) {
	db := setupDiscoveryTestDB(t)
	seedDiscoveryFixture(t, db)
	service := newDiscoveryTestService(t, db)

	result, err := service.discoveryDiscoverToolsHandler(devModeContext(), newDiscoveryCallRequest(nil))
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned an error result: %s", discoveryResultText(t, result))
	}

	var payload struct {
		Guidance   string              `json:"guidance"`
		Domains    []ServerToolSummary `json:"domains"`
		TotalTools int64               `json:"total_tools"`
	}
	if err := json.Unmarshal([]byte(discoveryResultText(t, result)), &payload); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}
	if payload.Guidance == "" {
		t.Fatalf("no-argument result carries no guidance")
	}
	if payload.TotalTools != 3 {
		t.Fatalf("total_tools = %d, want 3", payload.TotalTools)
	}
	if len(payload.Domains) != 2 || payload.Domains[0].Name != "jira" || payload.Domains[1].Name != "wiki" {
		t.Fatalf("unexpected domains: %+v", payload.Domains)
	}
}

func TestDiscoveryDiscoverTools_ServerSignatures(t *testing.T) {
	db := setupDiscoveryTestDB(t)
	seedDiscoveryFixture(t, db)
	service := newDiscoveryTestService(t, db)

	result, err := service.discoveryDiscoverToolsHandler(
		devModeContext(), newDiscoveryCallRequest(map[string]any{"domain": "jira"}),
	)
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned an error result: %s", discoveryResultText(t, result))
	}

	text := discoveryResultText(t, result)
	want := "jira__search(query: str, limit: Annotated[int, '1 <= limit', 'limit <= 100'] = None) -> any\n  Search issues in a project"
	if !strings.Contains(text, want) {
		t.Fatalf("signatures missing\nexpected: %s\nactual:\n%s", want, text)
	}
	if !strings.Contains(text, "jira__create_issue() -> any") {
		t.Fatalf("signatures missing jira__create_issue:\n%s", text)
	}
	if strings.Contains(text, "archived_tool") {
		t.Fatalf("disabled tool leaked into signatures:\n%s", text)
	}
}

func TestDiscoveryDiscoverTools_SchemaFormat(t *testing.T) {
	db := setupDiscoveryTestDB(t)
	seedDiscoveryFixture(t, db)
	service := newDiscoveryTestService(t, db)

	result, err := service.discoveryDiscoverToolsHandler(
		devModeContext(), newDiscoveryCallRequest(map[string]any{"domain": "wiki", "format": "schema"}),
	)
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}

	var payload struct {
		Domain string `json:"domain"`
		Tools  []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"tools"`
	}
	if err := json.Unmarshal([]byte(discoveryResultText(t, result)), &payload); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}
	if payload.Domain != "wiki" || len(payload.Tools) != 1 || payload.Tools[0].Name != "wiki__confluence_search" {
		t.Fatalf("unexpected schema listing: %+v", payload)
	}
}

func TestDiscoveryDiscoverTools_DomainNotFound(t *testing.T) {
	db := setupDiscoveryTestDB(t)
	seedDiscoveryFixture(t, db)
	service := newDiscoveryTestService(t, db)

	result, err := service.discoveryDiscoverToolsHandler(
		devModeContext(), newDiscoveryCallRequest(map[string]any{"domain": "old"}),
	)
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("disabled server must not be listable")
	}

	var payload struct {
		Code    string `json:"code"`
		Details struct {
			AvailableDomains []string `json:"available_domains"`
		} `json:"details"`
	}
	if err := json.Unmarshal([]byte(discoveryResultText(t, result)), &payload); err != nil {
		t.Fatalf("failed to decode error result: %v", err)
	}
	if payload.Code != "domain_not_found" {
		t.Fatalf("code = %q, want domain_not_found", payload.Code)
	}
	if len(payload.Details.AvailableDomains) != 2 {
		t.Fatalf("available domains = %v, want jira and wiki", payload.Details.AvailableDomains)
	}
}

func TestDiscoveryGetToolSchema(t *testing.T) {
	db := setupDiscoveryTestDB(t)
	seedDiscoveryFixture(t, db)
	service := newDiscoveryTestService(t, db)

	result, err := service.discoveryGetToolSchemaHandler(
		devModeContext(), newDiscoveryCallRequest(map[string]any{"tool_name": "jira__search"}),
	)
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned an error result: %s", discoveryResultText(t, result))
	}

	var payload struct {
		Name        string         `json:"name"`
		Domain      string         `json:"domain"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	}
	if err := json.Unmarshal([]byte(discoveryResultText(t, result)), &payload); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}
	if payload.Name != "jira__search" || payload.Domain != "jira" {
		t.Fatalf("unexpected identity: %+v", payload)
	}
	properties, _ := payload.Parameters["properties"].(map[string]any)
	if _, ok := properties["query"]; !ok {
		t.Fatalf("parameters lost the input schema: %+v", payload.Parameters)
	}
}

func TestDiscoveryGetToolSchema_NotFoundSuggestsSimilarName(t *testing.T) {
	db := setupDiscoveryTestDB(t)
	seedDiscoveryFixture(t, db)
	service := newDiscoveryTestService(t, db)

	result, err := service.discoveryGetToolSchemaHandler(
		devModeContext(), newDiscoveryCallRequest(map[string]any{"tool_name": "jira__serch"}),
	)
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("unknown tool must return an error result")
	}

	var payload struct {
		Code    string `json:"code"`
		Details struct {
			Suggestions []string `json:"suggestions"`
		} `json:"details"`
	}
	if err := json.Unmarshal([]byte(discoveryResultText(t, result)), &payload); err != nil {
		t.Fatalf("failed to decode error result: %v", err)
	}
	if payload.Code != "tool_not_found" {
		t.Fatalf("code = %q, want tool_not_found", payload.Code)
	}
	if len(payload.Details.Suggestions) == 0 || payload.Details.Suggestions[0] != "jira__search" {
		t.Fatalf("suggestions = %v, want jira__search first", payload.Details.Suggestions)
	}
}

func TestDiscoveryExecuteTool_Validation(t *testing.T) {
	db := setupDiscoveryTestDB(t)
	seedDiscoveryFixture(t, db)
	service := newDiscoveryTestService(t, db)

	t.Run("missing tool name", func(t *testing.T) {
		result, err := service.discoveryExecuteToolHandler(devModeContext(), newDiscoveryCallRequest(nil))
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		var payload struct {
			Code string `json:"code"`
		}
		if jsonErr := json.Unmarshal([]byte(discoveryResultText(t, result)), &payload); jsonErr != nil {
			t.Fatalf("failed to decode error result: %v", jsonErr)
		}
		if !result.IsError || payload.Code != "invalid_arguments" {
			t.Fatalf("result = %+v, want invalid_arguments error", payload)
		}
	})

	t.Run("unknown tool", func(t *testing.T) {
		result, err := service.discoveryExecuteToolHandler(
			devModeContext(), newDiscoveryCallRequest(map[string]any{"tool_name": "jira__missing"}),
		)
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		var payload struct {
			Code string `json:"code"`
		}
		if jsonErr := json.Unmarshal([]byte(discoveryResultText(t, result)), &payload); jsonErr != nil {
			t.Fatalf("failed to decode error result: %v", jsonErr)
		}
		if !result.IsError || payload.Code != "tool_not_found" {
			t.Fatalf("result = %+v, want tool_not_found error", payload)
		}
	})
}

func TestDiscovery_EnterpriseAccessFiltering(t *testing.T) {
	db := setupDiscoveryTestDB(t)
	seedDiscoveryFixture(t, db)
	service := newDiscoveryTestService(t, db)

	client := &model.McpClient{
		Name:        "limited",
		AccessToken: "token",
		AllowList:   datatypes.JSON(`["wiki"]`),
	}
	ctx := enterpriseClientContext(client)

	t.Run("domain listing hides unauthorized servers", func(t *testing.T) {
		result, err := service.discoveryDiscoverToolsHandler(ctx, newDiscoveryCallRequest(nil))
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		var payload struct {
			Domains []ServerToolSummary `json:"domains"`
		}
		if jsonErr := json.Unmarshal([]byte(discoveryResultText(t, result)), &payload); jsonErr != nil {
			t.Fatalf("failed to decode result: %v", jsonErr)
		}
		if len(payload.Domains) != 1 || payload.Domains[0].Name != "wiki" {
			t.Fatalf("domains = %+v, want only wiki", payload.Domains)
		}
	})

	t.Run("unauthorized server listing is reported as not found", func(t *testing.T) {
		result, err := service.discoveryDiscoverToolsHandler(
			ctx, newDiscoveryCallRequest(map[string]any{"domain": "jira"}),
		)
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		var payload struct {
			Code    string `json:"code"`
			Details struct {
				AvailableDomains []string `json:"available_domains"`
			} `json:"details"`
		}
		if jsonErr := json.Unmarshal([]byte(discoveryResultText(t, result)), &payload); jsonErr != nil {
			t.Fatalf("failed to decode error result: %v", jsonErr)
		}
		if !result.IsError || payload.Code != "domain_not_found" {
			t.Fatalf("result = %+v, want domain_not_found error", payload)
		}
		if len(payload.Details.AvailableDomains) != 1 || payload.Details.AvailableDomains[0] != "wiki" {
			t.Fatalf("available_domains = %v, want only wiki", payload.Details.AvailableDomains)
		}
	})

	t.Run("unauthorized schema lookup is forbidden", func(t *testing.T) {
		result, err := service.discoveryGetToolSchemaHandler(
			ctx, newDiscoveryCallRequest(map[string]any{"tool_name": "jira__search"}),
		)
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		var payload struct {
			Code string `json:"code"`
		}
		if jsonErr := json.Unmarshal([]byte(discoveryResultText(t, result)), &payload); jsonErr != nil {
			t.Fatalf("failed to decode error result: %v", jsonErr)
		}
		if !result.IsError || payload.Code != "forbidden" {
			t.Fatalf("result = %+v, want forbidden error", payload)
		}
	})

	t.Run("available domains in not-found errors are filtered too", func(t *testing.T) {
		result, err := service.discoveryDiscoverToolsHandler(
			ctx, newDiscoveryCallRequest(map[string]any{"domain": "nope"}),
		)
		if err != nil {
			t.Fatalf("handler error = %v", err)
		}
		var payload struct {
			Details struct {
				AvailableDomains []string `json:"available_domains"`
			} `json:"details"`
		}
		if jsonErr := json.Unmarshal([]byte(discoveryResultText(t, result)), &payload); jsonErr != nil {
			t.Fatalf("failed to decode error result: %v", jsonErr)
		}
		if len(payload.Details.AvailableDomains) != 1 || payload.Details.AvailableDomains[0] != "wiki" {
			t.Fatalf("available_domains = %v, want only wiki", payload.Details.AvailableDomains)
		}
	})
}
