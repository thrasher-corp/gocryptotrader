package repository

import (
	"github.com/thrasher-corp/gocryptotrader/database"
)

func GetSQLDialect() string {
	switch database.DB.Config.Driver {
	case "sqlite", "sqlite3":
		return "sqlite"
	case "psql", "postgresl", "postgresql":
		return "postgres"
	}
	return ""
}
