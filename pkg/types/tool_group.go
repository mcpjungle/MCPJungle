package types

// ToolGroup represents a group (collection) of MCP Tools.
// A group can contain a subset of all available tools in the MCPJungle system.
// This allows you to expose a limited set of tools to certain mcp clients.
type ToolGroup struct {
	// Name is the unique name of the tool group (mandatory).
	Name string `json:"name"`
	// IncludedTools is a list of tools included in this group (mandatory).
	IncludedTools []string `json:"included_tools"`

	Description string `json:"description"`
}

// ToolGroupEndpoints contains the endpoints a MCP client can use to access a tool group.
type ToolGroupEndpoints struct {
	StreamableHTTPEndpoint string `json:"streamable_http_endpoint"`
	SSEEndpoint            string `json:"sse_endpoint"`
	SSEMessageEndpoint     string `json:"sse_message_endpoint"`
}

type CreateToolGroupResponse struct {
	*ToolGroupEndpoints
}

type GetToolGroupResponse struct {
	*ToolGroup
	*ToolGroupEndpoints
}
