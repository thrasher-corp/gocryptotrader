package repository

import (
	"github.com/thrasher-corp/gocryptotrader/database"
)

// GetSQLDialect returns current SQL Dialect based on enabled driver
func GetSQLDialect() string {
	cfg := database.DB.GetConfig()
	switch cfg.Driver {
	case "sqlite", "sqlite3":
		return database.DBSQLite3
	case "psql", "postgres", "postgresql":
		return database.DBPostgreSQL
	}
	return "invalid driver"
}
