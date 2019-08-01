package postgres

import (
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/thrasher-/gocryptotrader/database"
)

func Connect() (*database.Database, error) {
	connConfig := pgx.ConnConfig{
		Host:     database.Conn.Config.Host,
		Port:     database.Conn.Config.Port,
		User:     database.Conn.Config.Username,
		Password: database.Conn.Config.Password,
		Database: database.Conn.Config.Database,
	}

	connPool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     connConfig,
		AfterConnect:   nil,
		MaxConnections: 20,
		AcquireTimeout: 30 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	sqlxDB := stdlib.OpenDBFromPool(connPool)
	database.Conn.SQL = sqlx.NewDb(sqlxDB, "pgx")
	return database.Conn, nil
}
