package types

// ToolInputSchema defines the schema for the input parameters of a tool
type ToolInputSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

// Tool represents a tool provided by an MCP Server registered in the registry.
type Tool struct {
	Name        string          `json:"name"`
	Enabled     bool            `json:"enabled"`
	Description string          `json:"description"`
	InputSchema ToolInputSchema `json:"input_schema"`
}

// ToolInvokeResult represents the result of a Tool call.
// It is designed to be passed down to the end user.
type ToolInvokeResult struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	IsError bool           `json:"isError,omitempty"`

	Content           []map[string]any `json:"content"`
	StructuredContent any              `json:"structuredContent,omitempty"`
}
