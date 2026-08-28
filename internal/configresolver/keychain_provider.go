package configresolver

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"

	keyring "github.com/zalando/go-keyring"
)

type keychainProvider struct {
	currentUsername func() (string, error)
	get             func(service, account string) (string, error)
}

func newKeychainProvider() SecretProvider {
	return keychainProvider{
		currentUsername: currentUsername,
		get:             keyring.Get,
	}
}

func (p keychainProvider) Resolve(key string) (string, error) {
	account, err := p.currentUsername()
	if err != nil {
		return "", fmt.Errorf("failed to determine account for keychain secret %q: %w", key, err)
	}

	value, err := p.get(key, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("keychain secret %q was not found", key)
	}
	if err != nil {
		return "", fmt.Errorf("failed to resolve keychain secret %q: %w", key, err)
	}

	return value, nil
}

func currentUsername() (string, error) {
	for _, name := range []string{"USER", "USERNAME"} {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value, nil
		}
	}

	current, err := user.Current()
	if err != nil {
		return "", err
	}
	if current.Username == "" {
		return "", fmt.Errorf("current OS user has no username")
	}
	return current.Username, nil
}
