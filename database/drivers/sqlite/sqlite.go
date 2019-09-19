package sqlite

import (
	"database/sql"
	"path/filepath"

	// import sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database"
)

// Connect opens a connection to sqlite database and returns a pointer to database.DB
func Connect() (*database.Db, error) {
	if database.DB.Config.Database == "" {
		return nil, database.ErrNoDatabaseProvided
	}

	x := database.DB.Config.Database[len(database.DB.Config.Database)-3 : len(database.DB.Config.Database)]
	if x != ".db" {
		database.DB.Config.Database += ".db"
	}

	databaseFullLocation := filepath.Join(database.DB.DataPath, database.DB.Config.Database)

	dbConn, err := sql.Open("sqlite3", databaseFullLocation)
	if err != nil {
		return nil, err
	}

	database.DB.SQL = dbConn
	return database.DB, nil
}
