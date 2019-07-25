package postgresql

import (
	"time"

	"github.com/thrasher-/gocryptotrader/db"
	"github.com/thrasher-/gocryptotrader/db/drivers"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

func ConnectPSQL(connStr drivers.ConnectionDetails) (*db.DBStruct, error) {
	connConfig := pgx.ConnConfig{
		Host:     connStr.Host,
		Port:     connStr.Port,
		User:     connStr.Username,
		Password: connStr.Password,
		Database: connStr.Database,
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
	db.DBConn.SQL = sqlx.NewDb(sqlxDB, "pgx")
	return db.DBConn, nil
}
