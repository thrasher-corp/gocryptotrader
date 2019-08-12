package database

import (
	"errors"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
)

// Database holds a pointer to sql connection, DataPath which is used for file based databases
// and a pointer to a Config struct
type Database struct {
	Config   *Config
	DataPath string
	SQL      *sqlx.DB

	Connected bool
	Mu        sync.RWMutex
}

// Config holds connection information about the database what the driver type is and if its enabled or not
type Config struct {
	Enabled                   bool   `json:"enabled"`
	Driver                    string `json:"driver"`
	drivers.ConnectionDetails `json:"connectionDetails"`
}

// Conn is a global copy of Database{} struct
var Conn = &Database{}

var (
	ErrNoDatabaseProvided = errors.New("no database provided")
)
