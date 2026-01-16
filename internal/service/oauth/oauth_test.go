package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestCreateClient(t *testing.T) {
	tests := []struct {
		name        string
		client      *model.OAuthClient
		wantError   bool
		expectID    bool
		description string
	}{
		{
			name: "auto-generate client ID",
			client: &model.OAuthClient{
				Name:         "test-client",
				RedirectURIs: "https://example.com/callback",
				Description:  "Test OAuth client",
			},
			wantError: false,
			expectID:  true,
		},
		{
			name: "duplicate client ID should fail",
			client: &model.OAuthClient{
				ClientID:     "duplicate-id",
				Name:         "client-1",
				RedirectURIs: "https://example.com/callback",
			},
			wantError: false,
			expectID:  false,
		},
	}

	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	svc := NewOAuthService(setup.DB)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := svc.CreateClient(tt.client)
			if tt.wantError {
				testhelpers.AssertError(t, err)
				return
			}
			testhelpers.AssertNoError(t, err)
			testhelpers.AssertNotNil(t, created)
			testhelpers.AssertEqual(t, tt.client.Name, created.Name)
			testhelpers.AssertEqual(t, tt.client.RedirectURIs, created.RedirectURIs)
			if tt.expectID && created.ClientID == "" {
				t.Error("Expected ClientID to be auto-generated")
			}
		})
	}

	// Test duplicate client ID
	client2 := &model.OAuthClient{
		ClientID:     "duplicate-id",
		Name:         "client-2",
		RedirectURIs: "https://example.com/callback",
	}
	_, err := svc.CreateClient(client2)
	testhelpers.AssertError(t, err)
}

func TestCreateAuthorizeCode(t *testing.T) {
	tests := []struct {
		name                string
		clientID            string
		redirectURI         string
		scope               string
		codeChallenge       string
		codeChallengeMethod string
		wantPKCE            bool
	}{
		{
			name:                "without PKCE",
			clientID:            "test-client-id",
			redirectURI:         "https://example.com/callback",
			scope:               "mcp",
			codeChallenge:       "",
			codeChallengeMethod: "",
			wantPKCE:            false,
		},
		{
			name:                "with PKCE S256",
			clientID:            "test-client-id",
			redirectURI:         "https://example.com/callback",
			scope:               "mcp",
			codeChallenge:       "Bhy6PJ1TNLW8pgFd6JIYMXgzl0akTrq2oHTvPqtPkBM",
			codeChallengeMethod: "S256",
			wantPKCE:            true,
		},
	}

	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	svc := NewOAuthService(setup.DB)

	user := &model.User{
		Username:    "testuser",
		AccessToken: "test-token-123",
	}
	setup.DB.Create(user)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := svc.CreateAuthorizeCode(
				context.Background(),
				tt.clientID,
				user.ID,
				tt.redirectURI,
				tt.scope,
				tt.codeChallenge,
				tt.codeChallengeMethod,
			)
			testhelpers.AssertNoError(t, err)
			if code == "" {
				t.Error("Expected authorization code to be generated")
			}

			var authCode model.OAuthCode
			err = setup.DB.Where("code = ?", code).First(&authCode).Error
			testhelpers.AssertNoError(t, err)
			testhelpers.AssertEqual(t, tt.clientID, authCode.ClientID)
			testhelpers.AssertEqual(t, user.ID, authCode.UserID)

			if tt.wantPKCE {
				testhelpers.AssertEqual(t, tt.codeChallenge, authCode.CodeChallenge)
				testhelpers.AssertEqual(t, tt.codeChallengeMethod, authCode.CodeChallengeMethod)
			}
		})
	}
}

func TestExchangeCodeForToken(t *testing.T) {
	tests := []struct {
		name          string
		usePKCE       bool
		codeVerifier  string
		wrongVerifier bool
		expireCode    bool
		wantError     bool
		expectedError error
		description   string
	}{
		{
			name:         "successful exchange without PKCE",
			usePKCE:      false,
			codeVerifier: "",
			wantError:    false,
		},
		{
			name:         "successful exchange with PKCE",
			usePKCE:      true,
			codeVerifier: "test-verifier-123",
			wantError:    false,
		},
		{
			name:          "PKCE verification failure",
			usePKCE:       true,
			codeVerifier:  "test-verifier-123",
			wrongVerifier: true,
			wantError:     true,
		},
		{
			name:          "expired code",
			usePKCE:       false,
			codeVerifier:  "",
			expireCode:    true,
			wantError:     true,
			expectedError: ErrExpiredCode,
		},
	}

	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	svc := NewOAuthService(setup.DB)

	user := &model.User{
		Username:    "testuser",
		AccessToken: "test-token-123",
	}
	setup.DB.Create(user)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var code string
			var challenge string
			var verifier string

			if tt.usePKCE {
				verifier = "test-verifier-123"
				h := sha256.New()
				h.Write([]byte(verifier))
				challenge = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h.Sum(nil))
			}

			code, _ = svc.CreateAuthorizeCode(
				context.Background(),
				"test-client-id",
				user.ID,
				"https://example.com/callback",
				"mcp",
				challenge,
				func() string {
					if tt.usePKCE {
						return "S256"
					}
					return ""
				}(),
			)

			if tt.expireCode {
				var authCode model.OAuthCode
				setup.DB.Where("code = ?", code).First(&authCode)
				authCode.ExpiresAt = time.Now().Add(-1 * time.Minute)
				setup.DB.Save(&authCode)
			}

			exchangeVerifier := tt.codeVerifier
			if tt.wrongVerifier {
				exchangeVerifier = "wrong-verifier"
			}

			token, err := svc.ExchangeCodeForToken(code, "test-client-id", exchangeVerifier)

			if tt.wantError {
				testhelpers.AssertError(t, err)
				if tt.expectedError != nil {
					testhelpers.AssertEqual(t, tt.expectedError, err)
				}
				return
			}

			testhelpers.AssertNoError(t, err)
			testhelpers.AssertNotNil(t, token)
			if token.Token == "" {
				t.Error("Expected access token to be generated")
			}
			if token.RefreshToken == "" {
				t.Error("Expected refresh token to be generated")
			}
			testhelpers.AssertEqual(t, user.ID, token.UserID)
			testhelpers.AssertEqual(t, "test-client-id", token.ClientID)

			// Verify code was deleted (one-time use)
			var authCode model.OAuthCode
			err = setup.DB.Where("code = ?", code).First(&authCode).Error
			if err == nil {
				t.Error("Expected authorization code to be deleted after exchange")
			}
		})
	}
}

func TestGetTokenByValue(t *testing.T) {
	tests := []struct {
		name        string
		expireToken bool
		wantError   bool
	}{
		{
			name:        "valid token",
			expireToken: false,
			wantError:   false,
		},
		{
			name:        "expired token",
			expireToken: true,
			wantError:   true,
		},
	}

	setup := testhelpers.SetupTestDB(t)
	defer setup.Cleanup()

	svc := NewOAuthService(setup.DB)

	user := &model.User{
		Username:    "testuser",
		AccessToken: "test-token-123",
	}
	setup.DB.Create(user)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _ := svc.CreateAuthorizeCode(
				context.Background(),
				"test-client-id",
				user.ID,
				"https://example.com/callback",
				"mcp",
				"",
				"",
			)
			token, _ := svc.ExchangeCodeForToken(code, "test-client-id", "")

			if tt.expireToken {
				var oauthToken model.OAuthToken
				setup.DB.Where("token = ?", token.Token).First(&oauthToken)
				oauthToken.ExpiresAt = time.Now().Add(-1 * time.Hour)
				setup.DB.Save(&oauthToken)
			}

			retrieved, err := svc.GetTokenByValue(token.Token)

			if tt.wantError {
				testhelpers.AssertError(t, err)
				return
			}

			testhelpers.AssertNoError(t, err)
			testhelpers.AssertNotNil(t, retrieved)
			testhelpers.AssertEqual(t, token.Token, retrieved.Token)
			testhelpers.AssertEqual(t, token.UserID, retrieved.UserID)
		})
	}
}

func TestVerifyPKCE(t *testing.T) {
	verifier := "test-verifier-123"
	h := sha256.New()
	h.Write([]byte(verifier))
	s256Challenge := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h.Sum(nil))

	tests := []struct {
		name      string
		challenge string
		method    string
		verifier  string
		want      bool
	}{
		{
			name:      "S256 with correct verifier",
			challenge: s256Challenge,
			method:    "S256",
			verifier:  verifier,
			want:      true,
		},
		{
			name:      "S256 with wrong verifier",
			challenge: s256Challenge,
			method:    "S256",
			verifier:  "wrong-verifier",
			want:      false,
		},
		{
			name:      "plain with matching challenge",
			challenge: "plain-challenge",
			method:    "plain",
			verifier:  "plain-challenge",
			want:      true,
		},
		{
			name:      "plain with non-matching challenge",
			challenge: "plain-challenge",
			method:    "plain",
			verifier:  "wrong-challenge",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := verifyPKCE(tt.challenge, tt.method, tt.verifier)
			if got != tt.want {
				t.Errorf("verifyPKCE() = %v, want %v", got, tt.want)
			}
		})
	}
}
