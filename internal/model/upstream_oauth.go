package model

import (
	"time"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// UpstreamOAuthPendingSession stores an in-progress OAuth authorization flow for
// an upstream MCP server registration.
type UpstreamOAuthPendingSession struct {
	gorm.Model

	SessionID string `json:"session_id" gorm:"uniqueIndex;not null"`

	ServerName string                   `json:"server_name" gorm:"index;not null"`
	Transport  types.McpServerTransport `json:"transport" gorm:"type:varchar(30);not null"`

	// ServerInput stores the original RegisterServerInput payload so registration
	// can be resumed after the OAuth callback completes.
	ServerInput datatypes.JSON `json:"server_input" gorm:"type:jsonb;not null"`

	Force bool `json:"force" gorm:"not null;default:false"`

	RedirectURI  string         `json:"redirect_uri"`
	ClientID     string         `json:"client_id"`
	ClientSecret string         `json:"client_secret"`
	Scopes       datatypes.JSON `json:"scopes" gorm:"type:jsonb"`

	State        string    `json:"state" gorm:"not null"`
	CodeVerifier string    `json:"code_verifier" gorm:"not null"`
	ExpiresAt    time.Time `json:"expires_at" gorm:"index;not null"`

	InitiatedBy string `json:"initiated_by"`
}

// UpstreamOAuthToken stores gateway-scoped OAuth credentials for a registered
// upstream MCP server.
type UpstreamOAuthToken struct {
	gorm.Model

	ServerName string                   `json:"server_name" gorm:"uniqueIndex;not null"`
	Transport  types.McpServerTransport `json:"transport" gorm:"type:varchar(30);not null"`

	ClientID     string         `json:"client_id"`
	ClientSecret string         `json:"client_secret"`
	RedirectURI  string         `json:"redirect_uri"`
	Scopes       datatypes.JSON `json:"scopes" gorm:"type:jsonb"`

	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}
