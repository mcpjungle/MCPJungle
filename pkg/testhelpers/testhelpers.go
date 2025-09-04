// Package testhelpers provides common testing utilities and assertion functions
// for the MCPJungle project.
package testhelpers

import (
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// CreateTestDB creates a test database using SQLite in-memory database
func CreateTestDB() (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
}

// AssertError asserts that an error occurred
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// AssertNoError asserts that no error occurred
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// AssertNotNil asserts that an object is not nil
func AssertNotNil(t *testing.T, obj any) {
	t.Helper()
	if obj == nil {
		t.Error("Expected not nil, got nil")
	}
}

// AssertEqual asserts that two values are equal
func AssertEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertTrue asserts that a condition is true
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Error(message)
	}
}

// AssertFalse asserts that a condition is false
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Error(message)
	}
}

// AssertStringContains asserts that a string contains a substring
func AssertStringContains(t *testing.T, str, substr string) {
	t.Helper()
	if !Contains(str, substr) {
		t.Errorf("Expected string '%s' to contain '%s'", str, substr)
	}
}

// AssertStringNotContains asserts that a string does not contain a substring
func AssertStringNotContains(t *testing.T, str, substr string) {
	t.Helper()
	if Contains(str, substr) {
		t.Errorf("Expected string '%s' to not contain '%s'", str, substr)
	}
}

// AssertSliceLength asserts that a slice has the expected length
func AssertSliceLength(t *testing.T, slice any, expectedLength int) {
	t.Helper()
	switch v := slice.(type) {
	case []any:
		if len(v) != expectedLength {
			t.Errorf("Expected slice length %d, got %d", expectedLength, len(v))
		}
	case []string:
		if len(v) != expectedLength {
			t.Errorf("Expected slice length %d, got %d", expectedLength, len(v))
		}
	case []int:
		if len(v) != expectedLength {
			t.Errorf("Expected slice length %d, got %d", expectedLength, len(v))
		}
	default:
		t.Error("Unsupported slice type for length assertion")
	}
}

// AssertMapContainsKey asserts that a map contains a specific key
func AssertMapContainsKey(t *testing.T, m map[string]any, key string) {
	t.Helper()
	if _, exists := m[key]; !exists {
		t.Errorf("Expected map to contain key '%s'", key)
	}
}

// AssertMapNotContainsKey asserts that a map does not contain a specific key
func AssertMapNotContainsKey(t *testing.T, m map[string]any, key string) {
	t.Helper()
	if _, exists := m[key]; exists {
		t.Errorf("Expected map to not contain key '%s'", key)
	}
}

// AssertPanic asserts that a function panics
func AssertPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected function to panic")
		}
	}()
	fn()
}

// AssertNoPanic asserts that a function does not panic
func AssertNoPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected no panic, but got: %v", r)
		}
	}()
	fn()
}

// CreateTestTable creates a test table for table-driven tests
func CreateTestTable[T any](tests []T) []T {
	return tests
}

// RunTableTests runs table-driven tests
func RunTableTests[T any](t *testing.T, tests []T, testFunc func(t *testing.T, test T)) {
	for i, tt := range tests {
		t.Run(testName(i, tt), func(t *testing.T) {
			testFunc(t, tt)
		})
	}
}

// Helper function to generate test names
func testName(index int, test any) string {
	if named, ok := test.(interface{ Name() string }); ok {
		return named.Name()
	}
	return fmt.Sprintf("test_%d", index)
}

// Contains checks if a string contains a substring
// This is a public function that can be used by other packages
func Contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

// containsSubstring is a helper function for Contains
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// FormatError formats error messages consistently
func FormatError(expected, actual any) string {
	return fmt.Sprintf("Expected %v, got %v", expected, actual)
}

// FormatSliceError formats slice error messages
func FormatSliceError(expected, actual any) string {
	return fmt.Sprintf("Expected %v, got %v", expected, actual)
}

// FormatMapError formats map error messages
func FormatMapError(expected, actual any) string {
	return fmt.Sprintf("Expected %v, got %v", expected, actual)
}
