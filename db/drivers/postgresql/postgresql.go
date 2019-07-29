package postgresql

import (
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/thrasher-/gocryptotrader/db"
)

func ConnectPSQL() (*db.Database, error) {
	connConfig := pgx.ConnConfig{
		Host:     db.Conn.Config.Host,
		Port:     db.Conn.Config.Port,
		User:     db.Conn.Config.Username,
		Password: db.Conn.Config.Password,
		Database: db.Conn.Config.Database,
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
	db.Conn.SQL = sqlx.NewDb(sqlxDB, "pgx")
	return db.Conn, nil
}
