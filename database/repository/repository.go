package repository

import (
	"github.com/thrasher-corp/gocryptotrader/database"
)

// GetSQLDialec returns current SQL Dialect based on enabled driver
func GetSQLDialect() string {
	switch database.DB.Config.Driver {
	case "sqlite", "sqlite3":
		return "sqlite"
	case "psql", "postgresl", "postgresql":
		return "postgres"
	}
	return "no driver found"
}
