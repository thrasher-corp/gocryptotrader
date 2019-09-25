package repository

import (
	"github.com/thrasher-corp/gocryptotrader/database"
)

// GetSQLDialect returns current SQL Dialect based on enabled driver
func GetSQLDialect() string {
	switch database.DB.Config.Driver {
	case "sqlite", "sqlite3":
		return "sqlite3"
	case "psql", "postgres", "postgresql":
		return "postgres"
	}
	return "invalid driver"
}
