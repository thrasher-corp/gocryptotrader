package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/log"
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
// to use SQLite, closing the connection it replaces. On error con is not set
// and stays the caller's to close
func (i *Instance) SetSQLiteConnection(con *sql.DB) error {
	if i == nil {
		return ErrNilInstance
	}
	if con == nil {
		return errNilSQL
	}
	con.SetMaxOpenConns(1)
	i.replaceSQL(con)
	return nil
}

// SetPostgresConnection safely sets the global database instance's connection
// to use Postgres, closing the connection it replaces. On error con is not set
// and stays the caller's to close
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
	con.SetMaxOpenConns(2)
	con.SetMaxIdleConns(1)
	con.SetConnMaxLifetime(time.Hour)
	i.replaceSQL(con)
	return nil
}

// replaceSQL installs con and closes the connection it replaces, whose pool would otherwise keep its connections and
// opener routine for good. A failed close is only logged, as con is in place either way
func (i *Instance) replaceSQL(con *sql.DB) {
	i.m.Lock()
	replaced := i.SQL
	i.SQL = con
	i.m.Unlock()
	if replaced == nil || replaced == con {
		return
	}
	if err := replaced.Close(); err != nil {
		log.Errorf(log.DatabaseMgr, "Failed to close replaced database connection: %v", err)
	}
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
	i.m.Lock()
	defer i.m.Unlock()
	if i.SQL == nil {
		return errNilSQL
	}

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
	i.m.RLock()
	defer i.m.RUnlock()
	if i.SQL == nil {
		return nil, errNilSQL
	}
	return i.SQL, nil
}
