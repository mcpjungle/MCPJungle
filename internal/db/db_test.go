package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)

				// Verify it's a valid GORM database instance
				sqlDB, err := db.DB()
				require.NoError(t, err)
				require.NotNil(t, sqlDB)

				// Test basic connectivity
				err = sqlDB.Ping()
				assert.NoError(t, err)

				// Close the connection
				err = sqlDB.Close()
				assert.NoError(t, err)
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
	require.NoError(t, err)
	require.NotNil(t, db)

	// Verify SQLite database file was created
	_, err = os.Stat("mcp.db")
	assert.NoError(t, err, "SQLite database file should be created")

	// Test database operations
	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Test ping
	err = sqlDB.Ping()
	assert.NoError(t, err)

	// Test basic query
	var result int
	err = db.Raw("SELECT 1").Scan(&result).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	// Close connection
	err = sqlDB.Close()
	assert.NoError(t, err)
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
	require.NoError(t, err)
	require.NotNil(t, db)

	// Verify logger configuration is set to Silent
	// This is harder to test directly, but we can verify the database works
	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Test that database operations work (indicating proper configuration)
	err = sqlDB.Ping()
	assert.NoError(t, err)

	// Test a simple query
	var result string
	err = db.Raw("SELECT 'test'").Scan(&result).Error
	assert.NoError(t, err)
	assert.Equal(t, "test", result)

	err = sqlDB.Close()
	assert.NoError(t, err)
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
	require.NoError(t, err)
	require.NotNil(t, db1)

	db2, err := NewDBConnection("")
	require.NoError(t, err)
	require.NotNil(t, db2)

	// Both should work
	sqlDB1, err := db1.DB()
	require.NoError(t, err)

	sqlDB2, err := db2.DB()
	require.NoError(t, err)

	// Test both connections
	err = sqlDB1.Ping()
	assert.NoError(t, err)

	err = sqlDB2.Ping()
	assert.NoError(t, err)

	// Close connections
	err = sqlDB1.Close()
	assert.NoError(t, err)

	err = sqlDB2.Close()
	assert.NoError(t, err)
}

func TestNewDBConnection_WithCustomPath(t *testing.T) {
	// Test with a custom SQLite path by setting working directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	defer func() {
		err = os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	// Test SQLite creation in temp directory
	db, err := NewDBConnection("")
	require.NoError(t, err)
	require.NotNil(t, db)

	// Verify database file was created in temp directory
	dbPath := filepath.Join(tempDir, "mcp.db")
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "SQLite database file should be created in temp directory")

	sqlDB, err := db.DB()
	require.NoError(t, err)

	err = sqlDB.Ping()
	assert.NoError(t, err)

	err = sqlDB.Close()
	assert.NoError(t, err)
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
			assert.Error(t, err)
			assert.Nil(t, db)
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
