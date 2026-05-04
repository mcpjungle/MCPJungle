package mcp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	mcpgoclient "github.com/mark3labs/mcp-go/client"
	mcpgotransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/apierrors"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// upstreamOAuthPendingSessionTTL defines how long a pending upstream OAuth
// registration can remain incomplete before operators need to restart the flow.
const upstreamOAuthPendingSessionTTL = 10 * time.Minute

var errUpstreamOAuthDCRUnsupported = errors.New("upstream OAuth provider does not support dynamic client registration")

func upstreamOAuthDCRUnsupportedUserError() error {
	return fmt.Errorf(
		"upstream OAuth provider does not support dynamic client registration, and mcpjungle does not yet support the OAuth client identification flow required by this provider. Hint: if you already have provider-issued OAuth client credentials, retry registration by supplying the oauth_client_id and oauth_client_secret in the MCP server configuration",
	)
}

// UpstreamOAuthAuthorizationPendingError signals that upstream registration
// must pause until an operator completes an OAuth authorization step.
type UpstreamOAuthAuthorizationPendingError struct {
	SessionID        string
	AuthorizationURL string
	ExpiresAt        time.Time
}

func (e *UpstreamOAuthAuthorizationPendingError) Error() string {
	return "upstream OAuth authorization required"
}

type upstreamOAuthTokenStore struct {
	db         *gorm.DB
	serverName string
	transport  types.McpServerTransport
}

// GetToken loads the currently stored upstream OAuth token for a server.
func (s *upstreamOAuthTokenStore) GetToken(ctx context.Context) (*mcpgotransport.Token, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var record model.UpstreamOAuthToken
	if err := s.db.WithContext(ctx).Where("server_name = ?", s.serverName).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, mcpgotransport.ErrNoToken
		}
		return nil, err
	}

	if record.AccessToken == "" {
		return nil, mcpgotransport.ErrNoToken
	}

	return &mcpgotransport.Token{
		AccessToken:  record.AccessToken,
		TokenType:    record.TokenType,
		RefreshToken: record.RefreshToken,
		Scope:        record.Scope,
		ExpiresAt:    record.ExpiresAt,
	}, nil
}

// SaveToken persists an upstream OAuth token while preserving any previously
// stored client registration metadata for the same server.
func (s *upstreamOAuthTokenStore) SaveToken(ctx context.Context, token *mcpgotransport.Token) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	record := &model.UpstreamOAuthToken{
		ServerName:   s.serverName,
		Transport:    s.transport,
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Scope:        token.Scope,
		ExpiresAt:    token.ExpiresAt,
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.UpstreamOAuthToken
		err := tx.Where("server_name = ?", s.serverName).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(record).Error
		}
		if err != nil {
			return err
		}

		existing.Transport = s.transport
		existing.AccessToken = token.AccessToken
		existing.TokenType = token.TokenType
		existing.RefreshToken = token.RefreshToken
		existing.Scope = token.Scope
		existing.ExpiresAt = token.ExpiresAt
		return tx.Save(&existing).Error
	})
}

// scopesToJSON converts a scope list into the JSON form stored in the DB.
func scopesToJSON(scopes []string) datatypes.JSON {
	if len(scopes) == 0 {
		return []byte("[]")
	}
	data, _ := json.Marshal(scopes)
	return data
}

// scopesFromJSON reads a stored scope list from the DB.
func scopesFromJSON(data datatypes.JSON) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var scopes []string
	if err := json.Unmarshal(data, &scopes); err != nil {
		return nil, err
	}
	return scopes, nil
}

// ScopesFromJSONForAPI exposes the shared scope-decoding helper to the API layer.
func ScopesFromJSONForAPI(data datatypes.JSON) ([]string, error) {
	return scopesFromJSON(data)
}

// createServerFromRegisterInput rebuilds a transport-specific McpServer model
// from a saved RegisterServerInput payload.
func createServerFromRegisterInput(input *types.RegisterServerInput) (*model.McpServer, error) {
	transport, err := types.ValidateTransport(input.Transport)
	if err != nil {
		return nil, err
	}
	sessionMode, err := types.ValidateSessionMode(input.SessionMode)
	if err != nil {
		return nil, err
	}

	switch transport {
	case types.TransportStreamableHTTP:
		return model.NewStreamableHTTPServer(
			input.Name,
			input.Description,
			input.URL,
			input.BearerToken,
			input.Headers,
			sessionMode,
		)
	case types.TransportStdio:
		return model.NewStdioServer(
			input.Name,
			input.Description,
			input.Command,
			input.Args,
			input.Env,
			sessionMode,
		)
	default:
		return model.NewSSEServer(
			input.Name,
			input.Description,
			input.URL,
			input.BearerToken,
			sessionMode,
		)
	}
}

// prepareOAuthConfig builds the mcp-go OAuth client configuration used for
// upstream authorization bootstrap and later token-backed runtime calls.
func prepareOAuthConfig(input *types.RegisterServerInput, tokenStore mcpgotransport.TokenStore) mcpgoclient.OAuthConfig {
	return mcpgoclient.OAuthConfig{
		ClientID:     input.OAuthClientID,
		ClientSecret: input.OAuthClientSecret,
		RedirectURI:  input.OAuthRedirectURI,
		Scopes:       input.OAuthScopes,
		TokenStore:   tokenStore,
		PKCEEnabled:  true,
	}
}

// persistOAuthTokenMetadata stores gateway-scoped OAuth client metadata for a
// registered upstream server without clobbering token values.
func (m *MCPService) persistOAuthTokenMetadata(ctx context.Context, serverName string, transport types.McpServerTransport, redirectURI, clientID, clientSecret string, scopes []string) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record model.UpstreamOAuthToken
		err := tx.Where("server_name = ?", serverName).First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			record = model.UpstreamOAuthToken{
				ServerName:   serverName,
				Transport:    transport,
				RedirectURI:  redirectURI,
				ClientID:     clientID,
				ClientSecret: clientSecret,
				Scopes:       scopesToJSON(scopes),
			}
			return tx.Create(&record).Error
		}
		if err != nil {
			return err
		}

		record.Transport = transport
		record.RedirectURI = redirectURI
		record.ClientID = clientID
		record.ClientSecret = clientSecret
		record.Scopes = scopesToJSON(scopes)
		return tx.Save(&record).Error
	})
}

// bootstrapUpstreamOAuth starts a two-phase registration flow after an
// unauthenticated upstream registration attempt received an HTTP 401.
func (m *MCPService) bootstrapUpstreamOAuth(ctx context.Context, input *types.RegisterServerInput, server *model.McpServer, force bool, initiatedBy string) error {
	if input.OAuthRedirectURI == "" {
		return fmt.Errorf(
			"upstream server requires OAuth authorization, but oauth_redirect_uri was not provided: %w",
			errors.Join(apierrors.ErrInvalidInput, apierrors.ErrUpstreamOAuthRequired),
		)
	}

	var (
		oauthErr     *mcpgoclient.OAuthAuthorizationRequiredError
		authURL      string
		clientID     = input.OAuthClientID
		clientSecret = input.OAuthClientSecret
	)

	switch server.Transport {
	case types.TransportStreamableHTTP:
		conf, err := server.GetStreamableHTTPConfig()
		if err != nil {
			return err
		}
		opts := prepareSHTTPClientOptions(server.Name, conf)
		c, err := mcpgoclient.NewOAuthStreamableHttpClient(conf.URL, prepareOAuthConfig(input, mcpgoclient.NewMemoryTokenStore()), opts...)
		if err != nil {
			return fmt.Errorf("failed to create OAuth HTTP client for MCP server: %w", err)
		}
		defer c.Close()
		_, err = initializeHTTPClient(ctx, c, conf.URL, m.mcpServerInitReqTimeoutSec)
		if err == nil {
			return m.finalizeMcpServerRegistration(ctx, server)
		}
		if !errors.As(err, &oauthErr) {
			return err
		}
	case types.TransportSSE:
		conf, err := server.GetSSEConfig()
		if err != nil {
			return err
		}
		c, err := mcpgoclient.NewOAuthSSEClient(conf.URL, prepareOAuthConfig(input, mcpgoclient.NewMemoryTokenStore()))
		if err != nil {
			return fmt.Errorf("failed to create OAuth SSE client for MCP server: %w", err)
		}
		defer c.Close()
		err = c.Start(ctx)
		if err == nil {
			initReq := defaultSSEInitializeRequest()
			_, err = c.Initialize(ctx, initReq)
		}
		if err == nil {
			return m.finalizeMcpServerRegistration(ctx, server)
		}
		if !errors.As(err, &oauthErr) {
			return err
		}
	default:
		return m.finalizeMcpServerRegistration(ctx, server)
	}

	oauthHandler := oauthErr.Handler
	if oauthHandler == nil {
		return fmt.Errorf("upstream server requires OAuth authorization but no OAuth handler was returned")
	}

	codeVerifier, err := mcpgoclient.GenerateCodeVerifier()
	if err != nil {
		return fmt.Errorf("failed to generate PKCE code verifier: %w", err)
	}
	codeChallenge := mcpgoclient.GenerateCodeChallenge(codeVerifier)
	state, err := mcpgoclient.GenerateState()
	if err != nil {
		return fmt.Errorf("failed to generate OAuth state: %w", err)
	}

	if oauthHandler.GetClientID() == "" {
		clientID, clientSecret, err := registerOAuthClientWithoutEmptyScope(ctx, oauthHandler, input, "mcpjungle-"+server.Name)
		if err != nil {
			if errors.Is(err, errUpstreamOAuthDCRUnsupported) {
				// TODO: Once MCPJungle supports additional OAuth client identification
				// strategies (for example Client ID Metadata Documents), revisit
				// this special-case error. At that point this branch may no longer
				// be needed because the flow should continue with the alternative
				// strategy instead of failing here.
				return upstreamOAuthDCRUnsupportedUserError()
			}
			return fmt.Errorf("failed to dynamically register OAuth client: %w", err)
		}
		input.OAuthClientID = clientID
		input.OAuthClientSecret = clientSecret
		oauthHandler, err = m.buildOAuthHandlerForRegisteredClient(ctx, server, input)
		if err != nil {
			return fmt.Errorf("failed to rebuild OAuth handler after dynamic client registration: %w", err)
		}
	}

	clientID = oauthHandler.GetClientID()
	clientSecret = oauthHandler.GetClientSecret()

	authURL, err = oauthHandler.GetAuthorizationURL(ctx, state, codeChallenge)
	if err != nil {
		return fmt.Errorf("failed to build OAuth authorization URL: %w", err)
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to serialize server registration input: %w", err)
	}

	sessionID, err := generateOAuthSessionID()
	if err != nil {
		return fmt.Errorf("failed to generate OAuth session ID: %w", err)
	}
	expiresAt := time.Now().Add(upstreamOAuthPendingSessionTTL)

	if err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("server_name = ?", server.Name).Delete(&model.UpstreamOAuthPendingSession{}).Error; err != nil {
			return err
		}

		session := &model.UpstreamOAuthPendingSession{
			SessionID:    sessionID,
			ServerName:   server.Name,
			Transport:    server.Transport,
			ServerInput:  inputJSON,
			Force:        force,
			RedirectURI:  input.OAuthRedirectURI,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       scopesToJSON(input.OAuthScopes),
			State:        state,
			CodeVerifier: codeVerifier,
			ExpiresAt:    expiresAt,
			InitiatedBy:  initiatedBy,
		}
		return tx.Create(session).Error
	}); err != nil {
		return fmt.Errorf("failed to persist upstream OAuth session: %w", err)
	}

	return &UpstreamOAuthAuthorizationPendingError{
		SessionID:        sessionID,
		AuthorizationURL: authURL,
		ExpiresAt:        expiresAt,
	}
}

// buildOAuthHandlerForRegisteredClient recreates an OAuth-capable upstream client
// after dynamic registration so subsequent authorization URL generation uses the
// newly issued client credentials.
func (m *MCPService) buildOAuthHandlerForRegisteredClient(ctx context.Context, server *model.McpServer, input *types.RegisterServerInput) (*mcpgotransport.OAuthHandler, error) {
	var oauthErr *mcpgoclient.OAuthAuthorizationRequiredError

	switch server.Transport {
	case types.TransportStreamableHTTP:
		conf, err := server.GetStreamableHTTPConfig()
		if err != nil {
			return nil, err
		}
		opts := prepareSHTTPClientOptions(server.Name, conf)
		c, err := mcpgoclient.NewOAuthStreamableHttpClient(conf.URL, prepareOAuthConfig(input, mcpgoclient.NewMemoryTokenStore()), opts...)
		if err != nil {
			return nil, err
		}
		defer c.Close()
		_, err = initializeHTTPClient(ctx, c, conf.URL, m.mcpServerInitReqTimeoutSec)
		if !errors.As(err, &oauthErr) {
			if err == nil {
				return nil, fmt.Errorf("unexpectedly initialized upstream server while rebuilding OAuth handler")
			}
			return nil, err
		}
	case types.TransportSSE:
		conf, err := server.GetSSEConfig()
		if err != nil {
			return nil, err
		}
		c, err := mcpgoclient.NewOAuthSSEClient(conf.URL, prepareOAuthConfig(input, mcpgoclient.NewMemoryTokenStore()))
		if err != nil {
			return nil, err
		}
		defer c.Close()
		err = c.Start(ctx)
		if err == nil {
			_, err = c.Initialize(ctx, defaultSSEInitializeRequest())
		}
		if !errors.As(err, &oauthErr) {
			if err == nil {
				return nil, fmt.Errorf("unexpectedly initialized upstream server while rebuilding OAuth handler")
			}
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported transport type for OAuth: %s", server.Transport)
	}

	if oauthErr == nil || oauthErr.Handler == nil {
		return nil, fmt.Errorf("failed to retrieve OAuth handler")
	}
	return oauthErr.Handler, nil
}

// registerOAuthClientWithoutEmptyScope performs dynamic client registration but
// omits the "scope" field entirely when no explicit scopes were configured.
func registerOAuthClientWithoutEmptyScope(
	ctx context.Context,
	handler *mcpgotransport.OAuthHandler,
	input *types.RegisterServerInput,
	clientName string,
) (string, string, error) {
	metadata, err := handler.GetServerMetadata(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get server metadata: %w", err)
	}
	if metadata.RegistrationEndpoint == "" {
		return "", "", errUpstreamOAuthDCRUnsupported
	}

	regRequest := map[string]any{
		"client_name":                clientName,
		"redirect_uris":              []string{input.OAuthRedirectURI},
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
	}
	if len(input.OAuthScopes) > 0 {
		regRequest["scope"] = joinScopes(input.OAuthScopes)
	}
	if input.OAuthClientSecret != "" {
		regRequest["token_endpoint_auth_method"] = "client_secret_basic"
	}

	reqBody, err := json.Marshal(regRequest)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal registration request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata.RegistrationEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", "", fmt.Errorf("failed to create registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send registration request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return "", "", errUpstreamOAuthDCRUnsupported
		}
		body, _ := io.ReadAll(resp.Body)
		var oauthErr struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description,omitempty"`
		}
		if err := json.Unmarshal(body, &oauthErr); err == nil && oauthErr.Error != "" {
			if oauthErr.ErrorDescription != "" {
				return "", "", fmt.Errorf("OAuth error: %s - %s", oauthErr.Error, oauthErr.ErrorDescription)
			}
			return "", "", fmt.Errorf("OAuth error: %s", oauthErr.Error)
		}
		return "", "", fmt.Errorf("registration request failed with status %d: %s", resp.StatusCode, body)
	}

	var regResponse struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&regResponse); err != nil {
		return "", "", fmt.Errorf("failed to decode registration response: %w", err)
	}
	return regResponse.ClientID, regResponse.ClientSecret, nil
}

func joinScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}
	result := scopes[0]
	for i := 1; i < len(scopes); i++ {
		result += " " + scopes[i]
	}
	return result
}

// generateOAuthSessionID creates a cryptographically random identifier for a
// pending upstream OAuth registration session.
func generateOAuthSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// CompleteUpstreamOAuthSession exchanges the callback code for tokens,
// finalizes upstream registration, and deletes the pending auth session.
func (m *MCPService) CompleteUpstreamOAuthSession(ctx context.Context, sessionID, code, state string) (*model.McpServer, error) {
	if sessionID == "" || code == "" || state == "" {
		return nil, fmt.Errorf("session_id, code and state are required: %w", apierrors.ErrInvalidInput)
	}

	var session model.UpstreamOAuthPendingSession
	if err := m.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("upstream OAuth session not found: %w", apierrors.ErrNotFound)
		}
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("upstream OAuth session expired: %w", apierrors.ErrInvalidInput)
	}
	if state != session.State {
		return nil, fmt.Errorf("upstream OAuth session state mismatch: %w", apierrors.ErrInvalidInput)
	}

	var input types.RegisterServerInput
	if err := json.Unmarshal(session.ServerInput, &input); err != nil {
		return nil, fmt.Errorf("failed to decode stored server registration input: %w", err)
	}

	input.OAuthRedirectURI = session.RedirectURI
	input.OAuthClientID = session.ClientID
	input.OAuthClientSecret = session.ClientSecret
	scopes, err := scopesFromJSON(session.Scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode stored OAuth scopes: %w", err)
	}
	input.OAuthScopes = scopes

	server, err := createServerFromRegisterInput(&input)
	if err != nil {
		return nil, err
	}

	if err := m.persistOAuthTokenMetadata(ctx, server.Name, server.Transport, input.OAuthRedirectURI, input.OAuthClientID, input.OAuthClientSecret, input.OAuthScopes); err != nil {
		return nil, fmt.Errorf("failed to persist OAuth client metadata: %w", err)
	}

	if err := m.processOAuthAuthorizationCode(ctx, server, &input, session.CodeVerifier, state, code); err != nil {
		return nil, err
	}

	if session.Force {
		if _, err := m.GetMcpServer(server.Name); err == nil {
			if err := m.DeregisterMcpServer(server.Name); err != nil {
				return nil, fmt.Errorf("failed to deregister existing server during OAuth completion: %w", err)
			}
		} else if !errors.Is(err, apierrors.ErrNotFound) {
			return nil, fmt.Errorf("failed to check for existing server during OAuth completion: %w", err)
		}
	}

	if err := m.finalizeMcpServerRegistration(ctx, server); err != nil {
		return nil, err
	}

	if err := m.db.WithContext(ctx).Unscoped().Delete(&session).Error; err != nil {
		return nil, fmt.Errorf("failed to delete completed upstream OAuth session: %w", err)
	}

	return server, nil
}

// processOAuthAuthorizationCode recreates the upstream OAuth handler and uses it
// to exchange the callback code for stored upstream tokens.
func (m *MCPService) processOAuthAuthorizationCode(ctx context.Context, server *model.McpServer, input *types.RegisterServerInput, codeVerifier, state, code string) error {
	tokenStore := &upstreamOAuthTokenStore{
		db:         m.db,
		serverName: server.Name,
		transport:  server.Transport,
	}

	var oauthErr *mcpgoclient.OAuthAuthorizationRequiredError

	switch server.Transport {
	case types.TransportStreamableHTTP:
		conf, err := server.GetStreamableHTTPConfig()
		if err != nil {
			return err
		}
		opts := prepareSHTTPClientOptions(server.Name, conf)
		c, err := mcpgoclient.NewOAuthStreamableHttpClient(conf.URL, prepareOAuthConfig(input, tokenStore), opts...)
		if err != nil {
			return fmt.Errorf("failed to create OAuth HTTP client for token exchange: %w", err)
		}
		defer c.Close()
		_, err = c.Initialize(ctx, defaultHTTPInitializeRequest(conf.URL))
		if !errors.As(err, &oauthErr) {
			if err == nil {
				return fmt.Errorf("unexpectedly initialized upstream server before OAuth token exchange")
			}
			return fmt.Errorf("failed to prepare OAuth HTTP handler: %w", err)
		}
	case types.TransportSSE:
		conf, err := server.GetSSEConfig()
		if err != nil {
			return err
		}
		c, err := mcpgoclient.NewOAuthSSEClient(conf.URL, prepareOAuthConfig(input, tokenStore))
		if err != nil {
			return fmt.Errorf("failed to create OAuth SSE client for token exchange: %w", err)
		}
		defer c.Close()
		err = c.Start(ctx)
		if !errors.As(err, &oauthErr) {
			if err == nil {
				_, err = c.Initialize(ctx, defaultSSEInitializeRequest())
			}
			if !errors.As(err, &oauthErr) {
				if err == nil {
					return fmt.Errorf("unexpectedly initialized upstream server before OAuth token exchange")
				}
				return fmt.Errorf("failed to prepare OAuth SSE handler: %w", err)
			}
		}
	default:
		return fmt.Errorf("OAuth completion is only supported for HTTP and SSE MCP servers: %w", apierrors.ErrInvalidInput)
	}

	if oauthErr == nil || oauthErr.Handler == nil {
		return fmt.Errorf("failed to retrieve OAuth handler for upstream server")
	}

	oauthErr.Handler.SetExpectedState(state)
	if err := oauthErr.Handler.ProcessAuthorizationResponse(ctx, code, state, codeVerifier); err != nil {
		return fmt.Errorf("failed to exchange OAuth authorization code for token: %w", err)
	}

	return nil
}

// GetUpstreamOAuthToken retrieves the stored gateway-scoped upstream OAuth credentials
// metadata and tokens for a registered upstream MCP server.
func (m *MCPService) GetUpstreamOAuthToken(serverName string) (*model.UpstreamOAuthToken, error) {
	return getStoredUpstreamOAuthToken(m.db, serverName)
}

// getStoredUpstreamOAuthToken loads the stored upstream OAuth token record for a server.
func getStoredUpstreamOAuthToken(db *gorm.DB, serverName string) (*model.UpstreamOAuthToken, error) {
	var record model.UpstreamOAuthToken
	if err := db.Where("server_name = ?", serverName).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("upstream OAuth token not found: %w", apierrors.ErrNotFound)
		}
		return nil, err
	}
	return &record, nil
}
