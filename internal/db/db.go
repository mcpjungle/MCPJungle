// Package db provides database functionality for the MCPJungle application.
package db

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TODO: Turn this into a singleton class.
// Only one database connection should be created and used throughout the application.

// NewDBConnection creates a new database connection based on the provided DSN.
// If the DSN is empty, it falls back to an embedded SQLite database at "./mcp.db".
func NewDBConnection(dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if dsn == "" {
		log.Println("[db] DATABASE_URL not set – falling back to embedded SQLite ./mcp.db")
		dialector = sqlite.Open("mcp.db?_busy_timeout=5000&_journal_mode=WAL")
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
