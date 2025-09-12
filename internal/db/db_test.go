package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

func TestNewDBConnection(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		expectError bool
		cleanup     func()
	}{
		{
			name:        "empty DSN should use SQLite fallback",
			dsn:         "",
			expectError: false,
			cleanup: func() {
				// Clean up SQLite database file
				if err := os.Remove("mcp.db"); err != nil && !os.IsNotExist(err) {
					t.Logf("Failed to clean up mcp.db: %v", err)
				}
				if err := os.Remove("mcp.db-wal"); err != nil && !os.IsNotExist(err) {
					t.Logf("Failed to clean up mcp.db-wal: %v", err)
				}
				if err := os.Remove("mcp.db-shm"); err != nil && !os.IsNotExist(err) {
					t.Logf("Failed to clean up mcp.db-shm: %v", err)
				}
			},
		},
		{
			name:        "invalid PostgreSQL DSN should return error",
			dsn:         "postgres://invalid:invalid@localhost:5432/invalid",
			expectError: true,
			cleanup:     func() {},
		},
		{
			name:        "malformed DSN should return error",
			dsn:         "invalid://dsn",
			expectError: true,
			cleanup:     func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cleanup before test
			tt.cleanup()

			db, err := NewDBConnection(tt.dsn)

			if tt.expectError {
				testhelpers.AssertError(t, err)
				if db != nil {
					t.Errorf("Expected db to be nil, got %v", db)
				}
			} else {
				testhelpers.AssertNoError(t, err)
				testhelpers.AssertNotNil(t, db)

				// Verify it's a valid GORM database instance
				sqlDB, err := db.DB()
				testhelpers.AssertNoError(t, err)
				testhelpers.AssertNotNil(t, sqlDB)

				// Test basic connectivity
				err = sqlDB.Ping()
				testhelpers.AssertNoError(t, err)

				// Close the connection
				err = sqlDB.Close()
				testhelpers.AssertNoError(t, err)
			}

			// Cleanup after test
			tt.cleanup()
		})
	}
}

func TestNewDBConnection_SQLiteFallback(t *testing.T) {
	// Ensure no existing database file
	cleanup := func() {
		if err := os.Remove("mcp.db"); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to clean up mcp.db: %v", err)
		}
		if err := os.Remove("mcp.db-wal"); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to clean up mcp.db-wal: %v", err)
		}
		if err := os.Remove("mcp.db-shm"); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to clean up mcp.db-shm: %v", err)
		}
	}

	cleanup()
	defer cleanup()

	// Test with empty DSN
	db, err := NewDBConnection("")
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, db)

	// Verify SQLite database file was created
	_, err = os.Stat("mcp.db")
	testhelpers.AssertNoError(t, err)

	// Test database operations
	sqlDB, err := db.DB()
	testhelpers.AssertNoError(t, err)

	// Test ping
	err = sqlDB.Ping()
	testhelpers.AssertNoError(t, err)

	// Test basic query
	var result int
	err = db.Raw("SELECT 1").Scan(&result).Error
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertEqual(t, 1, result)

	// Close connection
	err = sqlDB.Close()
	testhelpers.AssertNoError(t, err)
}

func TestNewDBConnection_DatabaseConfiguration(t *testing.T) {
	cleanup := func() {
		if err := os.Remove("mcp.db"); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to clean up mcp.db: %v", err)
		}
		if err := os.Remove("mcp.db-wal"); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to clean up mcp.db-wal: %v", err)
		}
		if err := os.Remove("mcp.db-shm"); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to clean up mcp.db-shm: %v", err)
		}
	}

	cleanup()
	defer cleanup()

	db, err := NewDBConnection("")
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, db)

	// Verify logger configuration is set to Silent
	// This is harder to test directly, but we can verify the database works
	sqlDB, err := db.DB()
	testhelpers.AssertNoError(t, err)

	// Test that database operations work (indicating proper configuration)
	err = sqlDB.Ping()
	testhelpers.AssertNoError(t, err)

	// Test a simple query
	var result string
	err = db.Raw("SELECT 'test'").Scan(&result).Error
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertEqual(t, "test", result)

	err = sqlDB.Close()
	testhelpers.AssertNoError(t, err)
}

func TestNewDBConnection_ConcurrentAccess(t *testing.T) {
	cleanup := func() {
		if err := os.Remove("mcp.db"); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to clean up mcp.db: %v", err)
		}
		if err := os.Remove("mcp.db-wal"); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to clean up mcp.db-wal: %v", err)
		}
		if err := os.Remove("mcp.db-shm"); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to clean up mcp.db-shm: %v", err)
		}
	}

	cleanup()
	defer cleanup()

	// Test creating multiple connections to the same SQLite database
	db1, err := NewDBConnection("")
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, db1)

	db2, err := NewDBConnection("")
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, db2)

	// Both should work
	sqlDB1, err := db1.DB()
	testhelpers.AssertNoError(t, err)

	sqlDB2, err := db2.DB()
	testhelpers.AssertNoError(t, err)

	// Test both connections
	err = sqlDB1.Ping()
	testhelpers.AssertNoError(t, err)

	err = sqlDB2.Ping()
	testhelpers.AssertNoError(t, err)

	// Close connections
	err = sqlDB1.Close()
	testhelpers.AssertNoError(t, err)

	err = sqlDB2.Close()
	testhelpers.AssertNoError(t, err)
}

func TestNewDBConnection_WithCustomPath(t *testing.T) {
	// Test with a custom SQLite path by setting working directory
	originalDir, err := os.Getwd()
	testhelpers.AssertNoError(t, err)

	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	testhelpers.AssertNoError(t, err)

	defer func() {
		err = os.Chdir(originalDir)
		testhelpers.AssertNoError(t, err)
	}()

	// Test SQLite creation in temp directory
	db, err := NewDBConnection("")
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, db)

	// Verify database file was created in temp directory
	dbPath := filepath.Join(tempDir, "mcp.db")
	_, err = os.Stat(dbPath)
	testhelpers.AssertNoError(t, err)

	sqlDB, err := db.DB()
	testhelpers.AssertNoError(t, err)

	err = sqlDB.Ping()
	testhelpers.AssertNoError(t, err)

	err = sqlDB.Close()
	testhelpers.AssertNoError(t, err)
}

func TestNewDBConnection_ErrorHandling(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
	}{
		{
			name: "invalid host",
			dsn:  "postgres://user:pass@invalidhost:5432/db",
		},
		{
			name: "invalid port",
			dsn:  "postgres://user:pass@localhost:99999/db",
		},
		{
			name: "invalid credentials",
			dsn:  "postgres://invaliduser:invalidpass@localhost:5432/db",
		},
		{
			name: "malformed URL",
			dsn:  "not-a-valid-url",
		},
		{
			name: "unsupported database",
			dsn:  "mysql://user:pass@localhost:3306/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDBConnection(tt.dsn)
			testhelpers.AssertError(t, err)
			if db != nil {
				t.Errorf("Expected db to be nil, got %v", db)
			}
		})
	}
}

// Benchmark tests
func BenchmarkNewDBConnection_SQLite(b *testing.B) {
	cleanup := func() {
		if err := os.Remove("mcp.db"); err != nil && !os.IsNotExist(err) {
			b.Logf("Failed to clean up mcp.db: %v", err)
		}
		if err := os.Remove("mcp.db-wal"); err != nil && !os.IsNotExist(err) {
			b.Logf("Failed to clean up mcp.db-wal: %v", err)
		}
		if err := os.Remove("mcp.db-shm"); err != nil && !os.IsNotExist(err) {
			b.Logf("Failed to clean up mcp.db-shm: %v", err)
		}
	}

	cleanup()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, err := NewDBConnection("")
		if err != nil {
			b.Fatal(err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			b.Fatal(err)
		}

		sqlDB.Close()
	}
}
