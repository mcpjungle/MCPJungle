package types

// ToolGroup represents a group (collection) of MCP Tools.
// A group contains a list of tools that are exposed
type ToolGroup struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	IncludedTools []string `json:"included_tools"`
}

type CreateToolGroupResponse struct {
	Endpoint string `json:"endpoint"`
}
