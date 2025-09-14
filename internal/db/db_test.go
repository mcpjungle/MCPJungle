package db

import (
	"os"
	"testing"
)

func TestNewDBConnection_SQLiteFallback(t *testing.T) {
	// Unset DATABASE_URL to force fallback
	os.Unsetenv("DATABASE_URL")
	db, err := NewDBConnection("")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if db == nil {
		t.Fatal("expected db instance, got nil")
	}
}

func TestNewDBConnection_PostgresDSN(t *testing.T) {
	// Use a fake DSN, expect connection error
	db, err := NewDBConnection("postgres://invalid:invalid@localhost:5432/invalid?sslmode=disable")
	if err == nil {
		t.Error("expected error for invalid DSN, got nil")
	}
	if db != nil {
		t.Error("expected nil db for invalid DSN")
	}
}
