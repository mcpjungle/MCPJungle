// Package db provides database functionality for the MCPJungle application.
package db

import (
	"fmt"
	"log"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const DB_SQLITE_FALLBACK_WARNING = "[db] DATABASE_URL not set – falling back to embedded SQLite ./mcp.db"
const DB_SQLITE_CONN_STRING = "mcp.db?_busy_timeout=5000&_journal_mode=WAL"

var db *gorm.DB
var err error
var once sync.Once

// NewDBConnection creates a new database connection based on the provided DSN.
// If the DSN is empty, it falls back to an embedded SQLite database at "./mcp.db".
func NewDBConnection(dsnString string) (*gorm.DB, error) {
	var dbErr error
	once.Do(func() {
		var dialector gorm.Dialector

		if dsnString == "" {
			log.Println(DB_SQLITE_FALLBACK_WARNING)
			dialector = sqlite.Open(DB_SQLITE_CONN_STRING)
		} else {
			dialector = postgres.Open(dsnString)
		}

		c := &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		}

		db, err = gorm.Open(dialector, c)

		if err != nil {
			dbErr = fmt.Errorf("failed to connect to database: %w", err)
		}
	})
	if dbErr != nil {
		return nil, dbErr
	}
	return db, err
}
