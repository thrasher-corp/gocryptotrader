package database

import (
	"context"
	"time"

	"github.com/thrasher-/gocryptotrader/database/base"
	"github.com/thrasher-/gocryptotrader/database/postgres"
	"github.com/thrasher-/gocryptotrader/database/sqlite3"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/gctrpc"
)

// Databaser enforces standard database functionality
type Databaser interface {
	Setup(*base.ConnDetails) error
	Connect() error
	UserLogin(newUser bool) error
	IsConnected() bool
	GetName() string
	InsertPlatformTrades(exchangeName string, trades []*base.PlatformTrades) error
	GetPlatformTradeLast(exchangeName, currencyPair, assetType string) (time.Time, string, error)
	GetPlatformTradeFirst(exchangeName, currencyPair, assetType string) (time.Time, string, error)
	GetFullPlatformHistory(exchName, currencyPair, assetType string) ([]*exchange.PlatformTrade, error)
	Disconnect() error

	// RPC specific  functions
	GetUserRPC(ctx context.Context, username string) (*base.User, error)
	InsertUserRPC(ctx context.Context, username, password string) error
	GetExchangeLoadedDataRPC(ctx context.Context, exchange string) ([]*gctrpc.AvailableData, error)
	GetExchangePlatformHistoryRPC(ctx context.Context, exchange, pair, asset string) ([]*gctrpc.PlatformHistory, error)
	GetUserAuditRPC(ctx context.Context, username string) ([]*base.Audit, error)
	GetUsersRPC(ctx context.Context) ([]*base.User, error)
	EnableDisableUserRPC(ctx context.Context, username string, enable bool) error
	SetUserPasswordRPC(ctx context.Context, username, password string) error
	ModifyUserRPC(ctx context.Context, username, email string) error
}

// GetSQLite3Instance returns a connection to a SQLite3 database
func GetSQLite3Instance() Databaser {
	return new(sqlite3.SQLite3)
}

// GetPostgresInstance returns a connection to a PostgreSQL database
func GetPostgresInstance() Databaser {
	return new(postgres.Postgres)
}
