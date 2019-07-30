package sqlite

import (
	"errors"
	"path/filepath"

	"github.com/jmoiron/sqlx"

	// import sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
	"github.com/thrasher-/gocryptotrader/database"
)

func Connect() (*database.Database, error) {
	if database.Conn.Config.Database == "" {
		return nil, errors.New("no database provided")
	}
	databaseFullLocation := filepath.Join(database.Conn.DataPath, database.Conn.Config.Database)
	dbConn, err := sqlx.Open("sqlite3", databaseFullLocation)
	if err != nil {
		return nil, err
	}

	database.Conn.SQL = dbConn
	return database.Conn, nil
}
