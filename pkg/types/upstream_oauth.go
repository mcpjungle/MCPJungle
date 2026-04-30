package types

import "time"

type UpstreamOAuthAuthorizationRequired struct {
	SessionID        string    `json:"session_id"`
	AuthorizationURL string    `json:"authorization_url"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type RegisterServerResult struct {
	Server                *McpServer                          `json:"server,omitempty"`
	AuthorizationRequired *UpstreamOAuthAuthorizationRequired `json:"authorization_required,omitempty"`
}

type CompleteUpstreamOAuthSessionInput struct {
	Code  string `json:"code"`
	State string `json:"state"`
}
