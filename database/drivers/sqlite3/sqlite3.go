package sqlite

import (
	"database/sql"
	"path/filepath"

	// import sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database"
)

// Connect opens a connection to sqlite database and returns a pointer to database.DB
func Connect() (*database.Instance, error) {
	cfg := database.DB.GetConfig()
	if cfg.Database == "" {
		return nil, database.ErrNoDatabaseProvided
	}

	databaseFullLocation := filepath.Join(database.DB.DataPath, cfg.Database)

	dbConn, err := sql.Open("sqlite3", databaseFullLocation)
	if err != nil {
		return nil, err
	}

	database.DB.SetSQLiteConnection(dbConn)

	return database.DB, nil
}
