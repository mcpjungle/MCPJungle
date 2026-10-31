package mcp

import (
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestProjectUIResourceURI(t *testing.T) {
	original := "ui://widget/v1.html"
	tool := mcpgo.Tool{
		Name: "render",
		Meta: &mcpgo.Meta{AdditionalFields: map[string]any{
			"ui":                map[string]any{"resourceUri": original, "prefersBorder": true},
			"openai/fileParams": []any{"file"},
		}},
	}
	projected := projectUIResourceURI("my-backend", tool)
	require.Equal(t, buildResourceURI("my-backend", original), ToolUIResourceURI(projected))
	require.Equal(t, original, ToolUIResourceURI(tool), "upstream metadata must stay untouched")
	require.Equal(t, []any{"file"}, projected.Meta.AdditionalFields["openai/fileParams"])
	require.Equal(t, true, projected.Meta.AdditionalFields["ui"].(map[string]any)["prefersBorder"])
	require.Equal(t, buildResourceURI("my-backend", original), ToolUIResourceURI(projectUIResourceURI("my-backend", projected)), "idempotent")
}

func TestProjectUIResourceURIScopedAndUnrelatedMetadata(t *testing.T) {
	for _, uri := range []string{"", "https://external.example/app", "file:///etc/passwd"} {
		tool := mcpgo.Tool{Meta: &mcpgo.Meta{AdditionalFields: map[string]any{
			"ui": map[string]any{"resourceUri": uri},
		}}}
		require.Equal(t, uri, ToolUIResourceURI(projectUIResourceURI("scope", tool)))
	}
	require.Empty(t, ToolUIResourceURI(mcpgo.Tool{}))
}

func TestUIResourceForToolFailsClosedForCrossBackendAndDisabled(t *testing.T) {
	db := setupTestDBWithResources(t)
	backend := createTestServer(t, db)
	resource := createTestResource(t, db, backend, "ui://widget/v1.html", "MCP App")
	service := &MCPService{db: db}
	allowed := projectUIResourceURI(backend.Name, mcpgo.Tool{
		Name: backend.Name + "__render",
		Meta: &mcpgo.Meta{AdditionalFields: map[string]any{
			"ui": map[string]any{"resourceUri": "ui://widget/v1.html"},
		}},
	})
	got, err := service.UIResourceForTool(allowed)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, resource.URI, got.URI)

	crossBackend := allowed
	crossBackend.Name = "unrelated__render"
	got, err = service.UIResourceForTool(crossBackend)
	require.NoError(t, err)
	require.Nil(t, got, "a group must never borrow a different backend's resource")

	resource.Enabled = false
	require.NoError(t, db.Save(resource).Error)
	got, err = service.UIResourceForTool(allowed)
	require.NoError(t, err)
	require.Nil(t, got, "disabled UI assets must not be projected")
}
