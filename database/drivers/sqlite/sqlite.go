package sqlite

import (
	"errors"
	"path/filepath"
	"runtime"

	"github.com/thrasher-/gocryptotrader/common"

	"github.com/jmoiron/sqlx"

	// import sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
	"github.com/thrasher-/gocryptotrader/database"
)

func Connect() (*database.Database, error) {
	if database.Conn.Config.Database == "" {
		return nil, errors.New("no database provided")
	}
	databaseDir := filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "/database")
	err := common.CreateDir(databaseDir)
	if err != nil {
		return nil, err
	}
	databaseFullLocation := filepath.Join(databaseDir, database.Conn.Config.Database)
	dbConn, err := sqlx.Open("sqlite3", databaseFullLocation)
	if err != nil {
		return nil, err
	}

	database.Conn.SQL = dbConn
	return database.Conn, nil
}
