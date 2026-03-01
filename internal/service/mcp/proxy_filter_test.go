package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestMcpProxyToolFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mode      model.ServerMode
		client    *model.McpClient
		tools     []mcp.Tool
		wantNames []string
	}{
		{
			name: "development mode returns all tools",
			mode: model.ModeDev,
			tools: []mcp.Tool{
				{Name: "time__get_current_time"},
				{Name: "deepwiki__search_wiki"},
			},
			wantNames: []string{"time__get_current_time", "deepwiki__search_wiki"},
		},
		{
			name: "enterprise mode filters unauthorized tools",
			mode: model.ModeEnterprise,
			client: &model.McpClient{
				Name:      "claude",
				AllowList: datatypes.JSON(`["time"]`),
			},
			tools: []mcp.Tool{
				{Name: "time__get_current_time"},
				{Name: "deepwiki__search_wiki"},
			},
			wantNames: []string{"time__get_current_time"},
		},
		{
			name: "enterprise mode wildcard allows all tools",
			mode: model.ModeEnterprise,
			client: &model.McpClient{
				Name:      "cursor",
				AllowList: datatypes.JSON(`["*"]`),
			},
			tools: []mcp.Tool{
				{Name: "time__get_current_time"},
				{Name: "deepwiki__search_wiki"},
			},
			wantNames: []string{"time__get_current_time", "deepwiki__search_wiki"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(context.Background(), "mode", tt.mode)
			if tt.client != nil {
				ctx = context.WithValue(ctx, "client", tt.client)
			}

			got := ProxyToolFilter(ctx, tt.tools)
			assert.Equal(t, tt.wantNames, toolNames(got))
		})
	}
}

func TestMcpProxyToolFilter_MissingModeInContext(t *testing.T) {
	t.Parallel()

	got := ProxyToolFilter(context.Background(), []mcp.Tool{
		{Name: "time__get_current_time"},
	})

	assert.Empty(t, got)
}

func TestMcpProxyToolFilter_InvalidModeTypeInContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "mode", "enterprise")
	got := ProxyToolFilter(ctx, []mcp.Tool{
		{Name: "time__get_current_time"},
	})

	assert.Empty(t, got)
}

func TestMcpProxyToolFilter_EnterpriseMissingClientInContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	got := ProxyToolFilter(ctx, []mcp.Tool{
		{Name: "time__get_current_time"},
	})

	assert.Empty(t, got)
}

func TestMcpProxyToolFilter_EnterpriseInvalidClientTypeInContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = context.WithValue(ctx, "client", "not-a-client")
	got := ProxyToolFilter(ctx, []mcp.Tool{
		{Name: "time__get_current_time"},
	})

	assert.Empty(t, got)
}

func TestMcpProxyToolFilter_EnterpriseNilClientInContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	var client *model.McpClient
	ctx = context.WithValue(ctx, "client", client)
	got := ProxyToolFilter(ctx, []mcp.Tool{
		{Name: "time__get_current_time"},
	})

	assert.Empty(t, got)
}

func TestMcpProxyToolFilter_EnterpriseMalformedToolNamesAreDenied(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "mode", model.ModeEnterprise)
	ctx = context.WithValue(ctx, "client", &model.McpClient{
		Name:      "claude",
		AllowList: datatypes.JSON(`["time"]`),
	})

	got := ProxyToolFilter(ctx, []mcp.Tool{
		{Name: "missing_separator"},
		{Name: "time__get_current_time"},
	})

	assert.Equal(t, []string{"time__get_current_time"}, toolNames(got))
}

func toolNames(tools []mcp.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	return names
}
