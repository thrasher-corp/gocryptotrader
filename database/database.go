package database

import (
	"time"

	"github.com/thrasher-/gocryptotrader/database/base"
	"github.com/thrasher-/gocryptotrader/database/postgres"
	"github.com/thrasher-/gocryptotrader/database/sqlite3"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Databaser enforces standard database functionality
type Databaser interface {
	Setup(base.ConnDetails) error
	Connect() error
	ClientLogin(newClient bool) error
	IsConnected() bool
	GetClientDetails() (string, error)
	GetName() string
	InsertPlatformTrade(orderID,
		exchangeName,
		currencyPair,
		assetType,
		orderType string,
		amount,
		rate float64,
		fulfilledOn time.Time) error
	GetPlatformTradeLast(exchangeName,
		currencyPair,
		assetType string) (time.Time, string, error)
	GetFullPlatformHistory(exchName,
		currencyPair,
		assetType string) ([]exchange.PlatformTrade, error)
	Disconnect() error
}

// GetSQLite3Instance returns a connection to a SQLite3 database
func GetSQLite3Instance() Databaser {
	return new(sqlite3.SQLite3)
}

// GetPostgresInstance returns a connection to a PostgreSQL database
func GetPostgresInstance() Databaser {
	return new(postgres.Postgres)
}
