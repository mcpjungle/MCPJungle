package types

// APIErrorResponse is the JSON payload returned by MCPJungle HTTP endpoints
// when a request fails.
type APIErrorResponse struct {
	// Error is a human-readable error message describing what went wrong. Mandatory.
	Error string `json:"error"`
	// Code is an optional machine-readable error code for programmatic handling.
	Code string `json:"code,omitempty"`
}
