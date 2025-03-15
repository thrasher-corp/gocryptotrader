package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/database/drivers"
)

// Instance holds all information for a database instance
type Instance struct {
	SQL       *sql.DB
	DataPath  string
	config    *Config
	connected bool
	m         sync.RWMutex
}

// Config holds all database configurable options including enable/disabled & DSN settings
type Config struct {
	Enabled                   bool   `json:"enabled"`
	Verbose                   bool   `json:"verbose"`
	Driver                    string `json:"driver"`
	drivers.ConnectionDetails `json:"connectionDetails"`
}

var (
	// DB Global Database Connection
	DB = &Instance{}
	// MigrationDir which folder to look in for current migrations
	MigrationDir = filepath.Join("..", "..", "database", "migrations")
	// ErrNoDatabaseProvided error to display when no database is provided
	ErrNoDatabaseProvided = errors.New("no database provided")
	// ErrDatabaseSupportDisabled error to display when no database is provided
	ErrDatabaseSupportDisabled = errors.New("database support is disabled")
	// SupportedDrivers slice of supported database driver types
	SupportedDrivers = []string{DBSQLite, DBSQLite3, DBPostgreSQL}
	// ErrFailedToConnect for when a database fails to connect
	ErrFailedToConnect = errors.New("database failed to connect")
	// ErrDatabaseNotConnected for when a database is not connected
	ErrDatabaseNotConnected = errors.New("database is not connected")
	// DefaultSQLiteDatabase is the default sqlite3 database name to use
	DefaultSQLiteDatabase = "gocryptotrader.db"
	// ErrNilInstance for when a database is nil
	ErrNilInstance = errors.New("database instance is nil")
	// ErrNilConfig for when a config is nil
	ErrNilConfig  = errors.New("received nil config")
	errNilSQL     = errors.New("database SQL connection is nil")
	errFailedPing = errors.New("unable to verify database is connected, failed ping")
)

const (
	// DBSQLite const string for sqlite across code base
	DBSQLite = "sqlite"
	// DBSQLite3 const string for sqlite3 across code base
	DBSQLite3 = "sqlite3"
	// DBPostgreSQL const string for PostgreSQL across code base
	DBPostgreSQL = "postgres"
	// DBInvalidDriver const string for invalid driver
	DBInvalidDriver = "invalid driver"
)

// IDatabase allows for the passing of a database struct
// without giving the receiver access to all functionality
type IDatabase interface {
	IsConnected() bool
	GetSQL() (*sql.DB, error)
	GetConfig() *Config
}

// ISQL allows for the passing of a SQL connection
// without giving the receiver access to all functionality
type ISQL interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	Exec(string, ...any) (sql.Result, error)
	Query(string, ...any) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
