package internal

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateAccessToken(t *testing.T) {
	t.Run("successful token generation", func(t *testing.T) {
		token, err := GenerateAccessToken()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if token == "" {
			t.Fatal("Expected non-empty token")
		}
	})

	t.Run("token length validation", func(t *testing.T) {
		token, err := GenerateAccessToken()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Decode the base64 URL token to check actual byte length
		decoded, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(token)
		if err != nil {
			t.Fatalf("Expected valid base64 URL encoding, got error: %v", err)
		}

		expectedLength := 32 // 256 bits = 32 bytes
		if len(decoded) != expectedLength {
			t.Errorf("Expected decoded token length %d, got %d", expectedLength, len(decoded))
		}
	})

	t.Run("token uniqueness", func(t *testing.T) {
		// Generate multiple tokens and ensure they're unique
		tokens := make(map[string]bool)
		const numTokens = 100

		for i := 0; i < numTokens; i++ {
			token, err := GenerateAccessToken()
			if err != nil {
				t.Fatalf("Expected no error on token %d, got: %v", i, err)
			}

			if tokens[token] {
				t.Errorf("Duplicate token generated: %s", token)
			}
			tokens[token] = true
		}

		if len(tokens) != numTokens {
			t.Errorf("Expected %d unique tokens, got %d", numTokens, len(tokens))
		}
	})

	t.Run("token format validation", func(t *testing.T) {
		token, err := GenerateAccessToken()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check that token only contains valid base64 URL characters
		validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
		for _, char := range token {
			if !strings.ContainsRune(validChars, char) {
				t.Errorf("Token contains invalid character: %c", char)
			}
		}

		// Verify no padding characters (since we use NoPadding)
		if strings.Contains(token, "=") {
			t.Error("Token contains padding characters, expected none")
		}
	})

	t.Run("token entropy validation", func(t *testing.T) {
		// Generate multiple tokens and check for reasonable entropy
		tokens := make([]string, 50)
		for i := 0; i < 50; i++ {
			token, err := GenerateAccessToken()
			if err != nil {
				t.Fatalf("Expected no error on token %d, got: %v", i, err)
			}
			tokens[i] = token
		}

		// Check that tokens are reasonably different from each other
		// This is a basic check - in practice, crypto/rand should provide good entropy
		similarityCount := 0
		for i := 0; i < len(tokens); i++ {
			for j := i + 1; j < len(tokens); j++ {
				if tokens[i] == tokens[j] {
					similarityCount++
				}
			}
		}

		if similarityCount > 0 {
			t.Errorf("Found %d duplicate tokens, expected 0", similarityCount)
		}
	})
}

func TestGenerateAccessToken_ErrorHandling(t *testing.T) {
	// Note: It's difficult to test the actual error case of crypto/rand.Read failing
	// in a normal environment, but we can document the expected behavior

	t.Run("function signature validation", func(t *testing.T) {
		// Verify the function signature matches expectations
		// GenerateAccessToken is a function that should always be callable
		_ = GenerateAccessToken
	})

	t.Run("return type validation", func(t *testing.T) {
		token, err := GenerateAccessToken()

		// Both return values should be of correct types
		// token is already a string, so we just check it's not empty
		if token == "" {
			t.Error("First return value is empty string")
		}
		// err is already an error interface, so we just check it's nil
		if err != nil {
			t.Error("Expected no error, got error")
		}
	})
}

func TestGenerateAccessToken_EdgeCases(t *testing.T) {
	t.Run("multiple rapid calls", func(t *testing.T) {
		// Test that rapid successive calls work correctly
		const numRapidCalls = 1000
		tokens := make([]string, numRapidCalls)
		errors := make([]error, numRapidCalls)

		for i := 0; i < numRapidCalls; i++ {
			tokens[i], errors[i] = GenerateAccessToken()
		}

		// Check for any errors
		for i, err := range errors {
			if err != nil {
				t.Errorf("Error on rapid call %d: %v", i, err)
			}
		}

		// Check for any empty tokens
		for i, token := range tokens {
			if token == "" {
				t.Errorf("Empty token on rapid call %d", i)
			}
		}

		// Check for uniqueness
		tokenSet := make(map[string]bool)
		for _, token := range tokens {
			if tokenSet[token] {
				t.Errorf("Duplicate token found in rapid calls: %s", token)
			}
			tokenSet[token] = true
		}
	})

	t.Run("concurrent token generation", func(t *testing.T) {
		// Test that concurrent calls work correctly
		const numConcurrent = 100
		tokenChan := make(chan string, numConcurrent)
		errorChan := make(chan error, numConcurrent)

		// Launch concurrent goroutines
		for i := 0; i < numConcurrent; i++ {
			go func() {
				token, err := GenerateAccessToken()
				tokenChan <- token
				errorChan <- err
			}()
		}

		// Collect results
		tokens := make([]string, numConcurrent)
		errors := make([]error, numConcurrent)
		for i := 0; i < numConcurrent; i++ {
			tokens[i] = <-tokenChan
			errors[i] = <-errorChan
		}

		// Check for any errors
		for i, err := range errors {
			if err != nil {
				t.Errorf("Error on concurrent call %d: %v", i, err)
			}
		}

		// Check for any empty tokens
		for i, token := range tokens {
			if token == "" {
				t.Errorf("Empty token on concurrent call %d", i)
			}
		}

		// Check for uniqueness
		tokenSet := make(map[string]bool)
		for _, token := range tokens {
			if tokenSet[token] {
				t.Errorf("Duplicate token found in concurrent calls: %s", token)
			}
			tokenSet[token] = true
		}
	})
}
