package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/mcpjungle/mcpjungle/internal"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"gorm.io/gorm"
)

var (
	// ErrInvalidClient is returned when an OAuth client cannot be found or is invalid.
	ErrInvalidClient = errors.New("invalid client")

	// ErrExpiredCode is returned when an authorization code has expired.
	ErrExpiredCode = errors.New("expired code")

	// ErrInvalidCode is returned when an authorization code is invalid or not found.
	ErrInvalidCode = errors.New("invalid code")
)

// OAuthService provides methods to manage OAuth 2.0 clients, authorization codes, and tokens.
type OAuthService struct {
	db *gorm.DB
}

// NewOAuthService creates a new instance of OAuthService.
func NewOAuthService(db *gorm.DB) *OAuthService {
	return &OAuthService{db: db}
}

// CreateClient creates a new OAuth 2.0 client in the database.
// If no ClientID is provided, one will be auto-generated.
func (s *OAuthService) CreateClient(client *model.OAuthClient) (*model.OAuthClient, error) {
	// Auto-generate ClientID if not provided
	if client.ClientID == "" {
		id, err := internal.GenerateAccessToken()
		if err != nil {
			return nil, err
		}
		client.ClientID = id
	}
	if err := s.db.Create(client).Error; err != nil {
		return nil, fmt.Errorf("failed to create oauth client: %w", err)
	}
	return client, nil
}

// GetClient retrieves an OAuth client by its ClientID.
func (s *OAuthService) GetClient(clientID string) (*model.OAuthClient, error) {
	var client model.OAuthClient
	if err := s.db.Where("client_id = ?", clientID).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidClient
		}
		return nil, err
	}
	return &client, nil
}

// CreateAuthorizeCode generates a new authorization code for the OAuth 2.0 Authorization Code flow.
// This code is short-lived (10 minutes) and can only be used once.
func (s *OAuthService) CreateAuthorizeCode(ctx context.Context, clientID string, userID uint, redirectURI string, scope string, codeChallenge string, codeChallengeMethod string) (string, error) {
	// Generate a secure, random authorization code
	code, err := internal.GenerateAccessToken()
	if err != nil {
		return "", err
	}

	// Create the authorization code record with 10-minute expiration
	authCode := model.OAuthCode{
		Code:                code,
		ClientID:            clientID,
		UserID:              userID,
		RedirectURI:         redirectURI,
		Scope:               scope,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	}

	if err := s.db.Create(&authCode).Error; err != nil {
		return "", fmt.Errorf("failed to save authorization code: %w", err)
	}

	return code, nil
}

// ExchangeCodeForToken validates an authorization code and exchanges it for an access token.
// This implements the token endpoint of the OAuth 2.0 Authorization Code flow.
// The authorization code is single-use and will be deleted after successful exchange.
// PKCE verification is performed if a code challenge was provided during authorization.
// Returns an access token (24-hour lifetime) and a refresh token.
func (s *OAuthService) ExchangeCodeForToken(code string, clientID string, codeVerifier string) (*model.OAuthToken, error) {
	// Find the authorization code
	var authCode model.OAuthCode
	if err := s.db.Where("code = ? AND client_id = ?", code, clientID).First(&authCode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCode
		}
		return nil, err
	}

	// Delete the code once it's used (one-time use, even if exchange fails)
	defer s.db.Unscoped().Delete(&authCode)

	// Check if code has expired
	if time.Now().After(authCode.ExpiresAt) {
		return nil, ErrExpiredCode
	}

	// Verify PKCE if present (required for public clients like ChatGPT)
	if authCode.CodeChallenge != "" {
		if !verifyPKCE(authCode.CodeChallenge, authCode.CodeChallengeMethod, codeVerifier) {
			return nil, errors.New("pkce verification failed")
		}
	}

	// Generate access token (long-lived, 24 hours)
	tokenStr, err := internal.GenerateAccessToken()
	if err != nil {
		return nil, err
	}

	// Generate refresh token (for token renewal)
	refreshTokenStr, err := internal.GenerateAccessToken()
	if err != nil {
		return nil, err
	}

	// Create the token record
	token := model.OAuthToken{
		Token:        tokenStr,
		RefreshToken: refreshTokenStr,
		ClientID:     clientID,
		UserID:       authCode.UserID,
		Scope:        authCode.Scope,
		ExpiresAt:    time.Now().Add(24 * time.Hour), // 24h expiration
	}

	if err := s.db.Create(&token).Error; err != nil {
		return nil, fmt.Errorf("failed to save oauth token: %w", err)
	}

	return &token, nil
}

// GetTokenByValue retrieves an OAuth token by its value and validates that it hasn't expired.
func (s *OAuthService) GetTokenByValue(tokenStr string) (*model.OAuthToken, error) {
	var token model.OAuthToken
	if err := s.db.Where("token = ?", tokenStr).First(&token).Error; err != nil {
		return nil, err
	}
	// Check expiration
	if time.Now().After(token.ExpiresAt) {
		return nil, errors.New("token expired")
	}
	return &token, nil
}

// verifyPKCE verifies a PKCE (Proof Key for Code Exchange) challenge.
// PKCE is a security extension for OAuth 2.0 that prevents authorization code interception attacks.
func verifyPKCE(challenge, method, verifier string) bool {
	if method == "S256" {
		// Compute SHA256 hash of the verifier
		h := sha256.New()
		h.Write([]byte(verifier))
		// Base64URL encode without padding
		expected := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h.Sum(nil))
		return expected == challenge
	}
	// Plain method: direct comparison (less secure)
	return verifier == challenge
}

// ListClients retrieves all registered OAuth clients from the database.
func (s *OAuthService) ListClients() ([]model.OAuthClient, error) {
	var clients []model.OAuthClient
	if err := s.db.Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}

// DeleteClient removes an OAuth client from the database.
func (s *OAuthService) DeleteClient(clientID string) error {
	return s.db.Unscoped().Where("client_id = ?", clientID).Delete(&model.OAuthClient{}).Error
}
