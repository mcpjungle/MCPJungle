package configresolver

import (
	"errors"
	"strings"
	"testing"

	keyring "github.com/zalando/go-keyring"
)

func TestKeychainProviderUsesKeyAsServiceAndCurrentUserAsAccount(t *testing.T) {
	var gotService string
	var gotAccount string
	provider := keychainProvider{
		currentUsername: func() (string, error) { return "alice", nil },
		get: func(service, account string) (string, error) {
			gotService = service
			gotAccount = account
			return "secret-token", nil
		},
	}

	value, err := provider.Resolve("github-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "secret-token" {
		t.Fatalf("expected resolved secret, got %q", value)
	}
	if gotService != "github-token" {
		t.Fatalf("expected service %q, got %q", "github-token", gotService)
	}
	if gotAccount != "alice" {
		t.Fatalf("expected account %q, got %q", "alice", gotAccount)
	}
}

func TestKeychainProviderMissingSecret(t *testing.T) {
	provider := keychainProvider{
		currentUsername: func() (string, error) { return "alice", nil },
		get: func(_, _ string) (string, error) {
			return "", keyring.ErrNotFound
		},
	}

	_, err := provider.Resolve("missing-token")
	if err == nil || err.Error() != `keychain secret "missing-token" was not found` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKeychainProviderBackendErrorDoesNotExposeValue(t *testing.T) {
	provider := keychainProvider{
		currentUsername: func() (string, error) { return "alice", nil },
		get: func(_, _ string) (string, error) {
			return "must-not-appear", errors.New("keychain access denied")
		},
	}

	_, err := provider.Resolve("github-token")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), `failed to resolve keychain secret "github-token": keychain access denied`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "must-not-appear") {
		t.Fatalf("error exposed the secret value: %v", err)
	}
}

func TestKeychainProviderUsernameError(t *testing.T) {
	provider := keychainProvider{
		currentUsername: func() (string, error) { return "", errors.New("user lookup failed") },
		get: func(_, _ string) (string, error) {
			t.Fatal("keychain should not be queried when user lookup fails")
			return "", nil
		},
	}

	_, err := provider.Resolve("github-token")
	if err == nil || !strings.Contains(err.Error(), `failed to determine account for keychain secret "github-token": user lookup failed`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCurrentUsernamePrefersUserEnvironmentVariable(t *testing.T) {
	t.Setenv("USER", "alice")
	t.Setenv("USERNAME", "windows-alice")

	username, err := currentUsername()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "alice" {
		t.Fatalf("expected USER value, got %q", username)
	}
}

func TestCurrentUsernameFallsBackToUsernameEnvironmentVariable(t *testing.T) {
	t.Setenv("USER", "")
	t.Setenv("USERNAME", "windows-alice")

	username, err := currentUsername()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "windows-alice" {
		t.Fatalf("expected USERNAME value, got %q", username)
	}
}
