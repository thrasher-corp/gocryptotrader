package database

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/database/drivers"
)

type Db struct {
	SQL      *sql.DB
	DataPath string
	Config   *Config

	Connected bool
	Mu        sync.RWMutex
}

// Config holds all database configurable options includng enable/disabled & DSN settings
type Config struct {
	Enabled                   bool   `json:"enabled"`
	Driver                    string `json:"driver"`
	drivers.ConnectionDetails `json:"connectionDetails"`
}

var (
	// DB Global Database Connection
	DB = &Db{}

	wd, _ = os.Getwd()
	// MigrationDir which folder to look in for current migrations
	MigrationDir = filepath.Join(wd, "database", "migrations")

	// ErrNoDatabaseProvided error to display when no database is provided
	ErrNoDatabaseProvided = errors.New("no database provided")

	// SupportedDrivers slice of supported database driver types
	SupportedDrivers = []string{"sqlite3", "sqlite", "postgres"}

	// DefaultSQLiteDatabase is the default sqlite database name to use
	DefaultSQLiteDatabase = "gocryptotrader.db"
)
