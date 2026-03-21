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

// ResourceReadRequest represents a request to read a resource, optionally scoped to a server.
type ResourceReadRequest struct {
	URI    string `json:"uri"`
	Server string `json:"server,omitempty"`
}

// ResourceReadResult represents the result of reading a resource.
type ResourceReadResult struct {
	Contents []map[string]any `json:"contents"`
}
