package sqlite

import (
	"errors"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thrasher-/gocryptotrader/database"
)

func Connect() (*database.Database, error) {
	if database.Conn.Config.Database == "" {
		return nil, errors.New("no database provided")
	}
	dbConn, err := sqlx.Open("sqlite3", database.Conn.Config.Database)
	if err != nil {
		return nil, err
	}

	database.Conn.SQL = dbConn
	return database.Conn, nil
}


