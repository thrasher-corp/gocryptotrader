package postgres

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"

	// import go libpq driver package
	_ "github.com/lib/pq"
	"github.com/thrasher-corp/gocryptotrader/database"
)

// Connect opens a connection to Postgres database and returns a pointer to database.DB
func Connect(cfg *database.Config) (*database.Instance, error) {
	if cfg == nil {
		return nil, database.ErrNilConfig
	}
	if !cfg.Enabled {
		return nil, database.ErrDatabaseSupportDisabled
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}

	host := net.JoinHostPort(cfg.Host, strconv.FormatUint(uint64(cfg.Port), 10))
	configDSN := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.Username,
		cfg.Password,
		host,
		cfg.Database,
		cfg.SSLMode)

	db, err := sql.Open(database.DBPostgreSQL, configDSN)
	if err != nil {
		return nil, err
	}
	err = database.DB.SetPostgresConnection(db)
	if err != nil {
		return nil, err
	}
	return database.DB, nil
}
