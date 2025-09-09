// Package internal provides internal utility functionality for the MCPJungle application.
package internal

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateAccessToken generates a 256-bit secure random access token for user authentication.
func GenerateAccessToken() (string, error) {
	const tokenLength = 32
	b := make([]byte, tokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate access token: %v", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}
