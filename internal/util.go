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

// ValidateAccessToken checks if the provided access token is valid and
// aligns with mcpjungle's security standards.
func ValidateAccessToken(token string) error {
	if len(token) < 16 || len(token) > 64 {
		return fmt.Errorf("access token length should be between 16 and 64 characters")
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
