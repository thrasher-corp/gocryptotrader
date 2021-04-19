package database

import (
	"database/sql"
	"time"

	"github.com/thrasher-corp/sqlboiler/boil"
)

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

func (i *Instance) SetSQliteConnection(con *sql.DB) {
	i.m.Lock()
	defer i.m.Unlock()
	i.SQL = con
	i.SQL.SetMaxOpenConns(1)
}

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

func (i *Instance) SetConnected(v bool) {
	i.m.Lock()
	i.connected = v
	i.m.Unlock()
}

func (i *Instance) CloseConnection() error {
	i.m.Lock()
	defer i.m.Unlock()
	return i.SQL.Close()
}

func (i *Instance) IsConnected() bool {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.connected
}

func (i *Instance) GetConfig() *Config {
	i.m.RLock()
	defer i.m.RUnlock()
	cpy := i.config
	return cpy
}

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
