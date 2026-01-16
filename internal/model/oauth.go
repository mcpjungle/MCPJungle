package model

import (
	"time"

	"gorm.io/gorm"
)

// OAuthClient represents an OAuth 2.0 client application that can request access to MCPJungle resources.
// This is used when external services (like ChatGPT) want to connect to MCPJungle using OAuth 2.0.
// The client must be registered before it can initiate the OAuth flow.
type OAuthClient struct {
	gorm.Model
	// ClientID is a unique identifier for the OAuth client. If not provided during creation,
	// it will be auto-generated. This is what external services use to identify themselves.
	ClientID string `json:"client_id" gorm:"uniqueIndex;not null"`

	// ClientSecret is optional and only used for confidential clients. For public clients
	// using PKCE (like ChatGPT), this can be empty.
	ClientSecret string `json:"client_secret"` // Optional for public clients/PKCE

	// Name is a human-readable name for the client (e.g., "ChatGPT", "Claude").
	Name string `json:"name" gorm:"not null"`
	// RedirectURIs contains the allowed redirect URIs where the authorization server
	// can send users after they authorize the client. For ChatGPT, this should be:
	// "https://chatgpt.com/connector_platform_oauth_redirect"
	RedirectURIs string `json:"redirect_uris"` // Comma-separated or JSON list

	// Description provides additional context about the client.
	Description string `json:"description"`
}

// OAuthCode represents a temporary authorization code issued during the OAuth 2.0 authorization flow.
// This code is short-lived (10 minutes) and can only be used once to exchange for an access token.
// It implements the Authorization Code flow with PKCE support for enhanced security.
type OAuthCode struct {
	gorm.Model
	// Code is the authorization code that will be exchanged for an access token.
	// It's a one-time use, short-lived token.
	Code string `gorm:"uniqueIndex;not null"`

	// ClientID identifies which OAuth client requested this authorization.
	ClientID string `gorm:"not null"`

	// UserID identifies the MCPJungle user who authorized the client.
	UserID uint `gorm:"not null"` // The user who authorized

	// ExpiresAt is when this code becomes invalid. Typically 10 minutes from creation.
	ExpiresAt time.Time `gorm:"not null"`

	// RedirectURI is the URI where the client should receive the authorization code.
	RedirectURI string

	// CodeChallenge is used for PKCE (Proof Key for Code Exchange) to prevent authorization
	// code interception attacks. It's the hashed value of the code_verifier.
	CodeChallenge string // For PKCE

	// CodeChallengeMethod specifies the hashing method used (e.g., "S256" or "plain").
	CodeChallengeMethod string

	// Scope contains the requested permissions (e.g., "mcp", "openid", "profile").
	Scope string
}

// OAuthToken represents an access token issued via the OAuth 2.0 flow.
// This token is what external services use to authenticate API requests to MCPJungle.
// Tokens are long-lived (24 hours) and can be refreshed using the refresh token.
type OAuthToken struct {
	gorm.Model
	// Token is the access token that clients use in the Authorization header.
	Token string `gorm:"uniqueIndex;not null"`

	// RefreshToken can be used to obtain a new access token when the current one expires.
	RefreshToken string `gorm:"uniqueIndex"`

	// ClientID identifies which OAuth client this token was issued to.
	ClientID string `gorm:"not null"`

	// UserID identifies the MCPJungle user associated with this token.
	UserID uint `gorm:"not null"`

	// ExpiresAt is when this token becomes invalid. Typically 24 hours from creation.
	ExpiresAt time.Time `gorm:"not null"`

	// Scope contains the permissions granted to this token.
	Scope string
}
