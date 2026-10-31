package toolgroup

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/mcp"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestUIResourceScopedToPermittedToolsAndRevokedOnGroupUpdate(t *testing.T) {
	db := setupInMemoryDB(t)
	backend, err := model.NewStdioServer("ui-backend", "synthetic UI source", "echo", nil, nil, "")
	require.NoError(t, err)
	require.NoError(t, db.Create(backend).Error)

	ui := "ui://synthetic/canary-v1.html"
	publicURI := "mcpj://res/ui-backend/" + base64.RawStdEncoding.EncodeToString([]byte(ui))
	toolMeta, err := json.Marshal(map[string]any{"ui": map[string]any{"resourceUri": ui}})
	require.NoError(t, err)
	for _, tool := range []model.Tool{
		{ServerID: backend.ID, Name: "render", InputSchema: []byte("{\"type\":\"object\"}"), Meta: toolMeta, Enabled: true},
		{ServerID: backend.ID, Name: "ordinary", InputSchema: []byte("{\"type\":\"object\"}"), Enabled: true},
	} {
		require.NoError(t, db.Create(&tool).Error)
	}
	require.NoError(t, db.Create(&model.Resource{
		ServerID: backend.ID, URI: publicURI, OriginalURI: ui,
		Name: "Safe UI", MIMEType: "text/html;profile=mcp-app", Enabled: true,
	}).Error)
	onlyUI := &model.ToolGroup{
		Name:          "ui-group",
		IncludedTools: datatypes.JSON([]byte("[\"ui-backend__render\"]")),
	}
	noUI := &model.ToolGroup{
		Name:          "no-ui-group",
		IncludedTools: datatypes.JSON([]byte("[\"ui-backend__ordinary\"]")),
	}
	require.NoError(t, db.Create(onlyUI).Error)
	require.NoError(t, db.Create(noUI).Error)

	svc := newTestMCPService(t, db)
	groups, err := NewToolGroupService(db, svc)
	require.NoError(t, err)
	uiGroup, ok := groups.GetToolGroupMCPServer("ui-group")
	require.True(t, ok)
	resources := groupResources(t, uiGroup)
	require.Len(t, resources, 1, "selected UI resource must be advertised")
	require.Contains(t, resources, publicURI)
	exposed := uiGroup.ListTools()["ui-backend__render"]
	require.NotNil(t, exposed)
	require.Equal(t, publicURI, mcp.ToolUIResourceURI(exposed.Tool))
	otherGroup, ok := groups.GetToolGroupMCPServer("no-ui-group")
	require.True(t, ok)
	require.Empty(t, groupResources(t, otherGroup), "unreferenced resource must not leak into a tool-only group")

	_, err = groups.UpdateToolGroup("ui-group", &model.ToolGroup{
		Name:          "ui-group",
		IncludedTools: datatypes.JSON([]byte("[\"ui-backend__ordinary\"]")),
	})
	require.NoError(t, err)
	require.Empty(t, groupResources(t, uiGroup), "revocation must remove the previously selected UI resource")

	_, err = groups.UpdateToolGroup("ui-group", &model.ToolGroup{
		Name:          "ui-group",
		IncludedTools: datatypes.JSON([]byte("[\"ui-backend__render\"]")),
	})
	require.NoError(t, err)
	require.Contains(t, groupResources(t, uiGroup), publicURI)
}

func groupResources(t *testing.T, srv *mcpserver.MCPServer) map[string]mcpgo.Resource {
	t.Helper()
	client, err := mcpclient.NewInProcessClient(srv)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, client.Start(ctx))
	init := mcpgo.InitializeRequest{}
	init.Params.ProtocolVersion = mcpgo.LATEST_PROTOCOL_VERSION
	init.Params.ClientInfo = mcpgo.Implementation{Name: "ui-projection-regression", Version: "1"}
	_, err = client.Initialize(ctx, init)
	require.NoError(t, err)
	defer client.Close()
	result, err := client.ListResources(ctx, mcpgo.ListResourcesRequest{})
	require.NoError(t, err)
	mapped := make(map[string]mcpgo.Resource)
	for _, resource := range result.Resources {
		mapped[resource.URI] = resource
	}
	return mapped
}
