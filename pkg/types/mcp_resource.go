package types

// Resource represents a resource provided by an MCP Server registered in the registry.
type Resource struct {
	URI         string         `json:"uri"`
	Name        string         `json:"name"`
	Enabled     bool           `json:"enabled"`
	Description string         `json:"description"`
	MIMEType    string         `json:"mime_type"`
	Annotations map[string]any `json:"annotations,omitempty"`
	Meta        map[string]any `json:"meta,omitempty"`
}
