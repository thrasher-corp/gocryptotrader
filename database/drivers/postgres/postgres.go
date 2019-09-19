package postgres

import (
	"database/sql"
	"fmt"
	"time"

	// shut up golint
	_ "github.com/lib/pq"
	"github.com/thrasher-corp/gocryptotrader/database"
)

// Connect opens a connection to Postgres database and returns a pointer to database.DB
func Connect() (*database.Db, error) {

	if database.DB.Config.SSLMode == "" {
		database.DB.Config.SSLMode = "disable"
	}

	configDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		database.DB.Config.Username,
		database.DB.Config.Password,
		database.DB.Config.Host,
		database.DB.Config.Port,
		database.DB.Config.Database,
		database.DB.Config.SSLMode)

	db, err := sql.Open("postgres", configDSN)
	if err != nil {
		return nil, err
	}

	database.DB.SQL = db
	database.DB.SQL.SetMaxOpenConns(2)
	database.DB.SQL.SetMaxIdleConns(1)
	database.DB.SQL.SetConnMaxLifetime(time.Hour)

	return database.DB, nil
}
