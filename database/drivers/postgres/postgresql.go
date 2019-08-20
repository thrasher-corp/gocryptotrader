package postgres

import (
	"fmt"
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/thrasher-corp/gocryptotrader/database"
)

// Connect establishes a connection pool to the database
func Connect() (*database.Database, error) {
	configDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s database=%s sslmode=%s",
		database.Conn.Config.Host,
		database.Conn.Config.Port,
		database.Conn.Config.Username,
		database.Conn.Config.Password,
		database.Conn.Config.Database,
		database.Conn.Config.SSLMode)

	connConfig, err := pgx.ParseDSN(configDSN)
	if err != nil {
		return nil, err
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
