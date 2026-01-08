// Package internal provides internal utility functionality for the MCPJungle application.
package internal

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"unicode"
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

// ValidateAccessToken checks if a user-provided access token is valid.
// It doesn't impose many conditions to allow flexibility.
// It is up to the user to follow best security practices when assigning access tokens.
func ValidateAccessToken(token string) error {
	if len(token) < 8 {
		return fmt.Errorf("access token should be at least 8 characters in length")
	}
	if hasWhitespace(token) {
		return fmt.Errorf("access token should not contain whitespace characters")
	}
	return nil
}

// hasWhitespace checks if the access token contains any whitespace characters.
func hasWhitespace(token string) bool {
	for _, r := range token {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}
