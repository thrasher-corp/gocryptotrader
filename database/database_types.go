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

type Config struct {
	Enabled                   bool   `json:"enabled"`
	Driver                    string `json:"driver"`
	drivers.ConnectionDetails `json:"connectionDetails"`
}

var (
	DB = &Db{}

	wd, _        = os.Getwd()
	MigrationDir = filepath.Join(wd, "database", "migrations")

	// ErrNoDatabaseProvided error to display when no database is provided
	ErrNoDatabaseProvided = errors.New("no database provided")

	// SupportedDrivers slice of supported database driver types
	SupportedDrivers = []string{"sqlite", "postgres"}

	// DefaultSQLiteDatabase is the default sqlite database name to use
	DefaultSQLiteDatabase = "gocryptotrader.db"
)
