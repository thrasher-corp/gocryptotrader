package postgres

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/stdlib"

	"github.com/jackc/pgx"
	"github.com/thrasher-corp/gocryptotrader/database"
)

func Connect() (*database.Db, error) {
	configDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s database=%s sslmode=%s",
		database.DB.Config.Host,
		database.DB.Config.Port,
		database.DB.Config.Username,
		database.DB.Config.Password,
		database.DB.Config.Database,
		database.DB.Config.SSLMode)

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

	database.DB.SQL = stdlib.OpenDBFromPool(connPool)

	return database.DB, nil
}
