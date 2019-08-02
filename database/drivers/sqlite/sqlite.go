package sqlite

import (
	"path/filepath"

	"github.com/jmoiron/sqlx"
	// import sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
	"github.com/thrasher-/gocryptotrader/database"
)

// Connect creates a connection to the entered database
// With SQLite the database is not created until first read/write

func Connect() (*database.Database, error) {
	if database.Conn.Config.Database == "" {
		return nil, database.ErrNoDatabaseProvided
	}

	databaseFullLocation := filepath.Join(database.Conn.DataPath, database.Conn.Config.Database)
	dbConn, err := sqlx.Open("sqlite3", databaseFullLocation)
	if err != nil {
		return nil, err
	}

	database.Conn.SQL = dbConn
	return database.Conn, nil
}
