package database

import (
	"database/sql"
	"time"

	"github.com/thrasher-corp/sqlboiler/boil"
)

// SetConfig safely sets the global database instance's config with some
// basic locks and checks
func (i *Instance) SetConfig(cfg *Config) error {
	if i == nil {
		return errNilInstance
	}
	if cfg == nil {
		return errNilConfig
	}
	i.m.Lock()
	i.config = cfg
	if i.config.Verbose {
		boil.DebugMode = true
		boil.DebugWriter = Logger{}
	} else {
		boil.DebugMode = false
	}
	i.m.Unlock()
	return nil
}

// SetSQLiteConnection safely sets the global database instance's connection
// to use SQLite
func (i *Instance) SetSQLiteConnection(con *sql.DB) {
	i.m.Lock()
	defer i.m.Unlock()
	i.SQL = con
	i.SQL.SetMaxOpenConns(1)
}

// SetPostgresConnection safely sets the global database instance's connection
// to use Postgres
func (i *Instance) SetPostgresConnection(con *sql.DB) error {
	if err := con.Ping(); err != nil {
		return err
	}
	i.m.Lock()
	defer i.m.Unlock()
	i.SQL = con
	i.SQL.SetMaxOpenConns(2)
	i.SQL.SetMaxIdleConns(1)
	i.SQL.SetConnMaxLifetime(time.Hour)
	return nil
}

// SetConnected safely sets the global database instance's connected
// status
func (i *Instance) SetConnected(v bool) {
	i.m.Lock()
	i.connected = v
	i.m.Unlock()
}

// CloseConnection safely disconnects the global database instance
func (i *Instance) CloseConnection() error {
	i.m.Lock()
	defer i.m.Unlock()
	return i.SQL.Close()
}

// IsConnected safely checks the SQL connection status
func (i *Instance) IsConnected() bool {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.connected
}

// GetConfig safely returns a copy of the config
func (i *Instance) GetConfig() *Config {
	i.m.RLock()
	defer i.m.RUnlock()
	cpy := i.config
	return cpy
}

// Ping pings the database
func (i *Instance) Ping() error {
	if i == nil {
		return errNilInstance
	}
	i.m.RLock()
	defer i.m.RUnlock()
	if i.SQL == nil {
		return errNilSQL
	}
	return i.SQL.Ping()
}

func (i *Instance) GetSQL() *sql.DB {
	if i == nil || !i.IsConnected() {
		return nil
	}
	i.m.Lock()
	defer i.m.Unlock()
	resp := i.SQL
	return resp
}
