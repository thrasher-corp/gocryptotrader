package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SetConfig safely sets the global database instance's config with some
// basic locks and checks
func (i *Instance) SetConfig(cfg *Config) error {
	if i == nil {
		return ErrNilInstance
	}
	if cfg == nil {
		return ErrNilConfig
	}
	i.m.Lock()
	i.config = cfg
	i.m.Unlock()
	return nil
}

// SetSQLiteConnection safely sets the global database instance's connection
// to use SQLite
func (i *Instance) SetSQLiteConnection(con *sql.DB) error {
	if i == nil {
		return ErrNilInstance
	}
	if con == nil {
		return errNilSQL
	}
	i.m.Lock()
	defer i.m.Unlock()
	i.SQL = con
	i.SQL.SetMaxOpenConns(1)
	return nil
}

// SetPostgresConnection safely sets the global database instance's connection
// to use Postgres
func (i *Instance) SetPostgresConnection(con *sql.DB) error {
	if i == nil {
		return ErrNilInstance
	}
	if con == nil {
		return errNilSQL
	}
	if err := con.PingContext(context.TODO()); err != nil {
		return fmt.Errorf("%w %s", errFailedPing, err)
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
	if i == nil {
		return
	}
	i.m.Lock()
	i.connected = v
	i.m.Unlock()
}

// CloseConnection safely disconnects the global database instance
func (i *Instance) CloseConnection() error {
	if i == nil {
		return ErrNilInstance
	}
	if i.SQL == nil {
		return errNilSQL
	}
	i.m.Lock()
	defer i.m.Unlock()

	return i.SQL.Close()
}

// IsConnected safely checks the SQL connection status
func (i *Instance) IsConnected() bool {
	if i == nil {
		return false
	}
	i.m.RLock()
	defer i.m.RUnlock()
	return i.connected
}

// GetConfig safely returns a copy of the config
func (i *Instance) GetConfig() *Config {
	if i == nil {
		return nil
	}
	i.m.RLock()
	defer i.m.RUnlock()
	cpy := i.config
	return cpy
}

// Ping pings the database
func (i *Instance) Ping() error {
	if i == nil {
		return ErrNilInstance
	}
	if !i.IsConnected() {
		return ErrDatabaseNotConnected
	}
	i.m.RLock()
	defer i.m.RUnlock()
	if i.SQL == nil {
		return errNilSQL
	}
	return i.SQL.PingContext(context.TODO())
}

// GetSQL returns the sql connection
func (i *Instance) GetSQL() (*sql.DB, error) {
	if i == nil {
		return nil, ErrNilInstance
	}
	if i.SQL == nil {
		return nil, errNilSQL
	}
	i.m.Lock()
	defer i.m.Unlock()
	resp := i.SQL
	return resp, nil
}
