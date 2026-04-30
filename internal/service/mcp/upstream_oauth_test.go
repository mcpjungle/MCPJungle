package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mcpgotransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestGenerateOAuthSessionID(t *testing.T) {
	t.Parallel()

	first, err := generateOAuthSessionID()
	testhelpers.AssertNoError(t, err)
	second, err := generateOAuthSessionID()
	testhelpers.AssertNoError(t, err)

	if len(first) != 64 {
		t.Fatalf("expected hex session ID length 64, got %d", len(first))
	}
	if first == second {
		t.Fatal("expected two generated session IDs to differ")
	}
}

func TestScopesJSONRoundTrip(t *testing.T) {
	t.Parallel()

	data := scopesToJSON([]string{"mcp.read", "tasks.read"})
	var raw []string
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("expected valid JSON array, got error: %v", err)
	}

	scopes, err := scopesFromJSON(data)
	testhelpers.AssertNoError(t, err)
	if len(scopes) != 2 || scopes[0] != "mcp.read" || scopes[1] != "tasks.read" {
		t.Fatalf("unexpected scopes round-trip result: %#v", scopes)
	}
}

func TestUpstreamOAuthTokenStore_SaveAndGetToken(t *testing.T) {
	t.Parallel()

	setup := testhelpers.SetupMCPTest(t)
	defer setup.Cleanup()

	record := &model.UpstreamOAuthToken{
		ServerName:   "todoist",
		Transport:    types.TransportStreamableHTTP,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "http://127.0.0.1:7777/oauth/callback",
		Scopes:       scopesToJSON([]string{"mcp.read"}),
	}
	testhelpers.AssertNoError(t, setup.DB.Create(record).Error)

	store := &upstreamOAuthTokenStore{
		db:         setup.DB,
		serverName: "todoist",
		transport:  types.TransportStreamableHTTP,
	}

	token := &mcpgotransport.Token{
		AccessToken:  "access-token",
		TokenType:    "Bearer",
		RefreshToken: "refresh-token",
		Scope:        "mcp.read",
		ExpiresAt:    time.Now().Add(5 * time.Minute).UTC(),
	}
	testhelpers.AssertNoError(t, store.SaveToken(context.Background(), token))

	saved, err := store.GetToken(context.Background())
	testhelpers.AssertNoError(t, err)
	if saved.AccessToken != token.AccessToken {
		t.Fatalf("expected access token %q, got %q", token.AccessToken, saved.AccessToken)
	}

	var persisted model.UpstreamOAuthToken
	testhelpers.AssertNoError(t, setup.DB.Where("server_name = ?", "todoist").First(&persisted).Error)
	if persisted.ClientID != "client-id" {
		t.Fatalf("expected existing client metadata to be preserved, got %q", persisted.ClientID)
	}
	if persisted.RefreshToken != "refresh-token" {
		t.Fatalf("expected refresh token to be updated, got %q", persisted.RefreshToken)
	}
}
