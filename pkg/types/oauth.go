package types

// OAuthClient represents an OAuth 2.0 client configuration for creation
type OAuthClient struct {
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Name         string `json:"name"`
	RedirectURIs string `json:"redirect_uris"`
	Description  string `json:"description"`
}

// CreateOAuthClientRequest is the request body for creating an OAuth client
type CreateOAuthClientRequest struct {
	Name         string `json:"name"`
	RedirectURIs string `json:"redirect_uris"`
	Description  string `json:"description,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}
