package types

// ToolGroup represents a group (collection) of MCP Tools.
// A group can contain a subset of all available tools in the MCPJungle system.
// This allows you to expose a limited set of tools to certain mcp clients.
type ToolGroup struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	IncludedTools []string `json:"included_tools"`
}

type CreateToolGroupResponse struct {
	Endpoint string `json:"endpoint"`
}
