package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/internal/service/oauth"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// oidcConfigurationHandler returns the OpenID Connect discovery document.
// This endpoint is required by OAuth 2.0/OIDC clients (like ChatGPT) to discover
// the authorization server's capabilities and endpoints.
// It follows the OIDC Discovery specification: https://openid.net/specs/openid-connect-discovery-1_0.html
func (s *Server) oidcConfigurationHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		baseURL := getBaseURL(c)
		config := gin.H{
			"issuer":                                baseURL,
			"authorization_endpoint":                baseURL + "/oauth/authorize",
			"token_endpoint":                        baseURL + "/oauth/token",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"scopes_supported":                      []string{"openid", "profile", "offline_access"},
			"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
			"code_challenge_methods_supported":      []string{"plain", "S256"},
		}
		c.JSON(http.StatusOK, config)
	}
}

// oauthProtectedResourceHandler returns the OAuth 2.0 protected resource metadata.
// This endpoint helps clients discover the resource server's authorization requirements.
// It's used by ChatGPT to understand what scopes are needed and where to get authorization.
func (s *Server) oauthProtectedResourceHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		baseURL := getBaseURL(c)
		metadata := gin.H{
			"resource":              baseURL,
			"authorization_servers": []string{baseURL},
			"scopes_supported":      []string{"mcp"},
		}
		c.JSON(http.StatusOK, metadata)
	}
}

// oauthAuthorizeHandler handles the OAuth 2.0 authorization endpoint.
// This implements the authorization step of the Authorization Code flow with PKCE.
// GET: Displays an authorization form where users can grant access to the OAuth client.
// POST: Processes the authorization, generates an authorization code, and redirects back to the client.
// The authorization code is then exchanged for an access token via the token endpoint.
func (s *Server) oauthAuthorizeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract OAuth 2.0 authorization request parameters
		clientID := c.Query("client_id")
		redirectURI := c.Query("redirect_uri")
		state := c.Query("state")                               // CSRF protection token
		codeChallenge := c.Query("code_challenge")              // PKCE challenge
		codeChallengeMethod := c.Query("code_challenge_method") // PKCE method (S256 or plain)
		scope := c.Query("scope")                               // Requested permissions

		// GET request: Show authorization form
		if c.Request.Method == http.MethodGet {
			// In a production app, this would show a proper login/consent page.
			// For MCPJungle, we use a simple HTML form where users enter their
			// username and CLI access token to authorize the OAuth client.
			html := fmt.Sprintf(`
				<html>
				<body>
					<h2>Authorize MCPJungle Access</h2>
					<form method="POST">
						<input type="hidden" name="client_id" value="%s">
						<input type="hidden" name="redirect_uri" value="%s">
						<input type="hidden" name="state" value="%s">
						<input type="hidden" name="code_challenge" value="%s">
						<input type="hidden" name="code_challenge_method" value="%s">
						<input type="hidden" name="scope" value="%s">
						<div>
							<label>Username:</label>
							<input type="text" name="username" required>
						</div>
						<div>
							<label>Access Token (from mcpjungle CLI):</label>
							<input type="password" name="access_token" required>
						</div>
						<button type="submit">Authorize</button>
					</form>
				</body>
				</html>
			`, clientID, redirectURI, state, codeChallenge, codeChallengeMethod, scope)
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
			return
		}

		// POST request: Process authorization
		username := c.PostForm("username")
		accessToken := c.PostForm("access_token")

		// Verify user credentials using their MCPJungle CLI access token
		user, err := s.userService.GetUserByAccessToken(accessToken)
		if err != nil || user.Username != username {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or access token"})
			return
		}

		// Generate authorization code (short-lived, single-use)
		code, err := s.oauthService.CreateAuthorizeCode(
			c.Request.Context(),
			c.PostForm("client_id"),
			user.ID,
			c.PostForm("redirect_uri"),
			c.PostForm("scope"),
			c.PostForm("code_challenge"),
			c.PostForm("code_challenge_method"),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate code"})
			return
		}

		// Redirect back to the OAuth client with the authorization code
		// The client will then exchange this code for an access token
		targetRedirectURI := c.PostForm("redirect_uri")
		if strings.Contains(targetRedirectURI, "?") {
			targetRedirectURI += "&"
		} else {
			targetRedirectURI += "?"
		}
		// Include the authorization code and state (for CSRF protection)
		targetRedirectURI += fmt.Sprintf("code=%s&state=%s", code, c.PostForm("state"))

		c.Redirect(http.StatusFound, targetRedirectURI)
	}
}

// oauthTokenHandler handles the OAuth 2.0 token endpoint.
// This exchanges an authorization code for an access token and refresh token.
// It implements the token exchange step of the Authorization Code flow with PKCE.
// The authorization code must be valid, not expired, and the PKCE verifier must match.
func (s *Server) oauthTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only support authorization_code grant type
		grantType := c.PostForm("grant_type")
		if grantType != "authorization_code" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported grant type"})
			return
		}

		// Extract token exchange parameters
		code := c.PostForm("code")                  // Authorization code from authorize endpoint
		clientID := c.PostForm("client_id")         // OAuth client identifier
		codeVerifier := c.PostForm("code_verifier") // PKCE verifier (original random string)

		// Exchange code for token (validates PKCE, expiration, etc.)
		token, err := s.oauthService.ExchangeCodeForToken(code, clientID, codeVerifier)
		if err != nil {
			status := http.StatusBadRequest
			// Invalid or expired codes should return 401
			if err == oauth.ErrInvalidCode || err == oauth.ErrExpiredCode {
				status = http.StatusUnauthorized
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}

		// Return token response according to OAuth 2.0 spec
		c.JSON(http.StatusOK, gin.H{
			"access_token":  token.Token,
			"refresh_token": token.RefreshToken,
			"token_type":    "Bearer",
			"expires_in":    int(token.ExpiresAt.Sub(token.CreatedAt).Seconds()),
			"scope":         token.Scope,
		})
	}
}

// getBaseURL constructs the base URL for the current request.
// It handles both direct HTTPS connections and proxied connections (e.g., via ngrok).
// The X-Forwarded-Proto header is checked to support reverse proxies.
func getBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	// Check for X-Forwarded-Proto (important for ngrok and other reverse proxies)
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}

// createOAuthClientHandler handles the creation of a new OAuth 2.0 client.
// This is an admin-only endpoint that allows registering external services (like ChatGPT)
// that want to connect to MCPJungle via OAuth.
// The client will receive a ClientID that it uses to initiate OAuth flows.
func (s *Server) createOAuthClientHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req types.CreateOAuthClientRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create the OAuth client model
		client := &model.OAuthClient{
			Name:         req.Name,
			RedirectURIs: req.RedirectURIs,
			Description:  req.Description,
			ClientID:     req.ClientID,
			ClientSecret: req.ClientSecret,
		}

		// Save to database (ClientID auto-generated if not provided)
		created, err := s.oauthService.CreateClient(client)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, created)
	}
}

// listOAuthClientsHandler handles listing all registered OAuth clients.
// This is an admin-only endpoint useful for managing OAuth integrations.
func (s *Server) listOAuthClientsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clients, err := s.oauthService.ListClients()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, clients)
	}
}

// deleteOAuthClientHandler handles deleting an OAuth client.
// This immediately revokes access for that client - any tokens issued to it become invalid.
// This is an admin-only endpoint.
func (s *Server) deleteOAuthClientHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.Param("client_id")
		if err := s.oauthService.DeleteClient(clientID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
