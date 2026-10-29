package mcp

import (
	"errors"
	"fmt"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
)

// ToolUIResourceURI returns a tool's MCP Apps resource reference without guessing
// a resource from tool names, descriptors, or unrelated metadata.
func ToolUIResourceURI(tool mcpgo.Tool) string {
	if tool.Meta == nil {
		return ""
	}
	ui, ok := tool.Meta.AdditionalFields["ui"].(map[string]any)
	if !ok {
		return ""
	}
	uri, _ := ui["resourceUri"].(string)
	return uri
}

// projectUIResourceURI rewrites only the generic MCP Apps UI resource reference.
// The upstream tool and its unrelated metadata are never changed in place.
func projectUIResourceURI(serverName string, tool mcpgo.Tool) mcpgo.Tool {
	uri := ToolUIResourceURI(tool)
	if !strings.HasPrefix(uri, "ui://") {
		return tool
	}
	meta := *tool.Meta
	meta.AdditionalFields = make(map[string]any, len(tool.Meta.AdditionalFields))
	for k, v := range tool.Meta.AdditionalFields {
		meta.AdditionalFields[k] = v
	}
	ui := make(map[string]any)
	for k, v := range tool.Meta.AdditionalFields["ui"].(map[string]any) {
		ui[k] = v
	}
	ui["resourceUri"] = buildResourceURI(serverName, uri)
	meta.AdditionalFields["ui"] = ui
	tool.Meta = &meta
	return tool
}

// UIResourceForTool returns a registered and enabled resource only if it is
// referenced by a tool from the same backend. Group projections must not leak
// other backend resources or unreferenced files as MCP Apps resources.
func (m *MCPService) UIResourceForTool(tool mcpgo.Tool) (*mcpgo.Resource, error) {
	serverName, _, ok := splitServerToolName(tool.GetName())
	if !ok {
		return nil, nil
	}
	uri := ToolUIResourceURI(tool)
	if !strings.HasPrefix(uri, resourceURIPrefix+serverName+"/") {
		return nil, nil
	}
	resource, err := m.GetResource(uri)
	if errors.Is(err, apierrors.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("look up UI resource: %w", err)
	}
	if !resource.Enabled || resource.Server.Name != serverName {
		return nil, nil
	}
	projected, err := convertResourceModelToMcpObject(resource)
	if err != nil {
		return nil, fmt.Errorf("reconstruct UI resource: %w", err)
	}
	return &projected, nil
}
