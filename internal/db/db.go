// Package db provides database functionality for the MCPJungle application.
package db

import (
	"fmt"
	"log"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TODO: Turn this into a singleton class.
// Only one database connection should be created and used throughout the application.

const (
	dbFilename           = "mcpjungle.db"
	deprecatedDBFilename = "mcp.db"
)

// getSQLiteDBPath determines which SQLite database file to use.
// It prioritizes the new mcpjungle.db file, but falls back to the old mcp.db file for backward compatibility.
func getSQLiteDBPath() string {
	// Check if the new database file exists
	if _, err := os.Stat(dbFilename); err == nil {
		return dbFilename
	}

	// Check if the old database file exists (backward compatibility)
	if _, err := os.Stat(deprecatedDBFilename); err == nil {
		log.Printf("[db] WARNING: Using deprecated database file '%s'. Please consider renaming it to '%s' for future compatibility.", deprecatedDBFilename, dbFilename)
		return deprecatedDBFilename
	}

	// Neither exists, use the new file name
	return dbFilename
}

// NewDBConnection creates a new database connection based on the provided DSN.
// If the DSN is empty, it falls back to an embedded SQLite database.
// For backward compatibility, it will use an existing "mcp.db" file if present,
// otherwise it creates/uses "mcpjungle.db".
func NewDBConnection(dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if dsn == "" {
		dbPath := getSQLiteDBPath()
		log.Printf("[db] DATABASE_URL not set – falling back to embedded SQLite ./%s", dbPath)
		dialector = sqlite.Open(fmt.Sprintf("%s?_busy_timeout=5000&_journal_mode=WAL", dbPath))
	} else {
		dialector = postgres.Open(dsn)
	}

	c := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	db, err := gorm.Open(dialector, c)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}
