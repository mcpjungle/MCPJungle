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

type CreateToolGroupResponse struct {
	Endpoint string `json:"endpoint"`
}

type GetToolGroupResponse struct {
	*ToolGroup

	// Endpoint is the URL MCP clients can connect to, to access this tool group's MCP server.
	Endpoint string `json:"endpoint"`
}
