package types

import "time"

type UpstreamOAuthAuthorizationRequired struct {
	SessionID        string    `json:"session_id"`
	AuthorizationURL string    `json:"authorization_url"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// RegisterServerResult represents the result of a mcp server registration attempt.
// It may include the registered server details and/or information about required upstream OAuth authorization.
type RegisterServerResult struct {
	// Server contains the details of the registered MCP server, if registration was successful.
	Server *McpServer `json:"server,omitempty"`
	// AuthorizationRequired contains information about required upstream OAuth authorization, if the server
	// needs to perform OAuth authorization before it can be registered successfully.
	AuthorizationRequired *UpstreamOAuthAuthorizationRequired `json:"authorization_required,omitempty"`
}

// CompleteUpstreamOAuthSessionInput represents the input required to complete an upstream OAuth session.
type CompleteUpstreamOAuthSessionInput struct {
	Code  string `json:"code"`
	State string `json:"state"`
}
