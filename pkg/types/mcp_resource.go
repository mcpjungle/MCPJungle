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

// ResourceGetRequest represents a request to fetch resource metadata.
type ResourceGetRequest struct {
	URI string `json:"uri"`
}

// ResourceReadRequest represents a request to read a resource.
type ResourceReadRequest struct {
	URI string `json:"uri"`
}

// ResourceReadResult represents the result of reading a resource.
type ResourceReadResult struct {
	Contents []map[string]any `json:"contents"`
}
