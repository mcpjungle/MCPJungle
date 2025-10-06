package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

// cleanupDBFiles removes both old and new database files and their associated WAL and SHM files
func cleanupDBFiles(t *testing.T) {
	dbFiles := []string{"mcp.db", "mcpjungle.db"}
	extensions := []string{"", "-wal", "-shm"}

	for _, dbFile := range dbFiles {
		for _, ext := range extensions {
			file := dbFile + ext
			if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
				t.Logf("Failed to clean up %s: %v", file, err)
			}
		}
	}
}

// cleanupDBFilesBenchmark is the same as cleanupDBFiles but for benchmark tests
func cleanupDBFilesBenchmark(b *testing.B) {
	dbFiles := []string{"mcp.db", "mcpjungle.db"}
	extensions := []string{"", "-wal", "-shm"}

	for _, dbFile := range dbFiles {
		for _, ext := range extensions {
			file := dbFile + ext
			if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
				b.Logf("Failed to clean up %s: %v", file, err)
			}
		}
	}
}

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
				cleanupDBFiles(t)
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
	// Ensure no existing database files
	cleanup := func() {
		cleanupDBFiles(t)
	}

	cleanup()
	defer cleanup()

	// Test with empty DSN
	db, err := NewDBConnection("")
	testhelpers.AssertNoError(t, err)
	testhelpers.AssertNotNil(t, db)

	// Verify SQLite database file was created (should be the new name)
	_, err = os.Stat("mcpjungle.db")
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
		cleanupDBFiles(t)
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
		cleanupDBFiles(t)
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
	dbPath := filepath.Join(tempDir, "mcpjungle.db")
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
		cleanupDBFilesBenchmark(b)
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

// Test backward compatibility and new database file functionality
func TestSQLiteDBPath_BackwardCompatibility(t *testing.T) {
	cleanup := func() {
		cleanupDBFiles(t)
	}

	tests := []struct {
		name           string
		setup          func()
		expectedDBFile string
		expectWarning  bool
	}{
		{
			name:           "no existing files - should create new file",
			setup:          func() {},
			expectedDBFile: "mcpjungle.db",
			expectWarning:  false,
		},
		{
			name: "old file exists - should use old file with warning",
			setup: func() {
				oldDB, err := os.Create("mcp.db")
				testhelpers.AssertNoError(t, err)
				oldDB.Close()
			},
			expectedDBFile: "mcp.db",
			expectWarning:  true,
		},
		{
			name: "new file exists - should use new file",
			setup: func() {
				newDB, err := os.Create("mcpjungle.db")
				testhelpers.AssertNoError(t, err)
				newDB.Close()
			},
			expectedDBFile: "mcpjungle.db",
			expectWarning:  false,
		},
		{
			name: "both files exist - should prefer new file",
			setup: func() {
				oldDB, err := os.Create("mcp.db")
				testhelpers.AssertNoError(t, err)
				oldDB.Close()

				newDB, err := os.Create("mcpjungle.db")
				testhelpers.AssertNoError(t, err)
				newDB.Close()
			},
			expectedDBFile: "mcpjungle.db",
			expectWarning:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup() // Clean up before each test
			tt.setup()

			// Test the path selection logic
			result := getSQLiteDBPath()
			testhelpers.AssertEqual(t, tt.expectedDBFile, result)

			// Test that the database connection actually works
			db, err := NewDBConnection("")
			testhelpers.AssertNoError(t, err)
			testhelpers.AssertNotNil(t, db)

			// Verify the expected database file exists
			_, err = os.Stat(tt.expectedDBFile)
			testhelpers.AssertNoError(t, err)

			// Test basic functionality
			sqlDB, err := db.DB()
			testhelpers.AssertNoError(t, err)
			err = sqlDB.Ping()
			testhelpers.AssertNoError(t, err)
			err = sqlDB.Close()
			testhelpers.AssertNoError(t, err)

			cleanup() // Clean up after each test
		})
	}
}
