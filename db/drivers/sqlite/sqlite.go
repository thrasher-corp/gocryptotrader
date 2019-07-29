package sqlite

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thrasher-/gocryptotrader/db"
)

func Connect() (*db.Database, error) {
	dbConn, err := sqlx.Open("sqlite3", db.Conn.Config.Database)
	if err != nil {
		return nil, err
	}

	db.Conn.SQL = dbConn
	return db.Conn, nil
}


func CreateTable() error {
	query := `
CREATE TABLE IF NOT EXISTS audit
(
    id INTEGER PRIMARY KEY,
    Type       varchar(255),
    Identifier varchar(255),
    Message    text,
    created_at timestamp default CURRENT_TIMESTAMP   
);`
	_, err := db.Conn.SQL.Exec(query)
	if err != nil {
		return err
	}

	return nil
}