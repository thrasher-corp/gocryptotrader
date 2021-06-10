package database

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	// import sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

func TestSetConfig(t *testing.T) {
	t.Parallel()
	inst := &Instance{}
	err := inst.SetConfig(&Config{Verbose: true})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	err = inst.SetConfig(nil)
	if !errors.Is(err, ErrNilConfig) {
		t.Errorf("received %v, expected %v", err, ErrNilConfig)
	}

	inst = nil
	err = inst.SetConfig(&Config{})
	if !errors.Is(err, ErrNilInstance) {
		t.Errorf("received %v, expected %v", err, ErrNilInstance)
	}
}

func TestSetSQLiteConnection(t *testing.T) {
	t.Parallel()
	inst := &Instance{}
	err := inst.SetSQLiteConnection(nil)
	if !errors.Is(err, errNilSQL) {
		t.Errorf("received %v, expected %v", err, errNilSQL)
	}

	err = inst.SetSQLiteConnection(&sql.DB{})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	inst = nil
	err = inst.SetSQLiteConnection(nil)
	if !errors.Is(err, ErrNilInstance) {
		t.Errorf("received %v, expected %v", err, ErrNilInstance)
	}
}

func TestSetPostgresConnection(t *testing.T) {
	// there is nothing actually requiring a postgres connection specifically
	// so this is testing the checks and the ability to set values
	// however, such settings would be bad for a sqlite connection irl
	t.Parallel()
	inst := &Instance{}
	databaseFullLocation := filepath.Join(DB.DataPath, "TestSetPostgresConnection")
	con, err := sql.Open("sqlite3", databaseFullLocation)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	err = inst.SetPostgresConnection(con)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	err = con.Close()
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	err = os.Remove(databaseFullLocation)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
}

func TestSetConnected(t *testing.T) {
	t.Parallel()
	inst := &Instance{}
	inst.SetConnected(true)
	if !inst.connected {
		t.Errorf("received %v, expected %v", false, true)
	}
	inst.SetConnected(false)
	if inst.connected {
		t.Errorf("received %v, expected %v", true, false)
	}
}

func TestCloseConnection(t *testing.T) {
	t.Parallel()
	inst := &Instance{}
	databaseFullLocation := filepath.Join(DB.DataPath, "TestCloseConnection")
	con, err := sql.Open("sqlite3", databaseFullLocation)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	err = inst.SetSQLiteConnection(con)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	err = inst.CloseConnection()
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
}

func TestIsConnected(t *testing.T) {
	t.Parallel()
	inst := &Instance{}

	inst.SetConnected(true)
	if !inst.IsConnected() {
		t.Errorf("received %v, expected %v", false, true)
	}
	inst.SetConnected(false)
	if inst.IsConnected() {
		t.Errorf("received %v, expected %v", true, false)
	}
}

func TestGetConfig(t *testing.T) {
	t.Parallel()
	inst := &Instance{}

	cfg := inst.GetConfig()
	if cfg != nil {
		t.Errorf("received %v, expected %v", cfg, nil)
	}

	err := inst.SetConfig(&Config{Enabled: true})
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	cfg = inst.GetConfig()
	if cfg == nil {
		t.Errorf("received %v, expected %v", cfg, &Config{Enabled: true})
	}
}

func TestPing(t *testing.T) {
	t.Parallel()
	inst := &Instance{}
	databaseFullLocation := filepath.Join(DB.DataPath, "TestPing")
	con, err := sql.Open("sqlite3", databaseFullLocation)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	err = inst.SetSQLiteConnection(con)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	inst.SetConnected(true)
	err = inst.Ping()
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	inst.SQL = nil
	err = inst.Ping()
	if !errors.Is(err, errNilSQL) {
		t.Errorf("received %v, expected %v", err, errNilSQL)
	}
	inst.SetConnected(false)
	err = inst.Ping()
	if !errors.Is(err, ErrDatabaseNotConnected) {
		t.Errorf("received %v, expected %v", err, ErrDatabaseNotConnected)
	}
	inst = nil
	err = inst.Ping()
	if !errors.Is(err, ErrNilInstance) {
		t.Errorf("received %v, expected %v", err, ErrNilInstance)
	}
	err = con.Close()
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	err = os.Remove(databaseFullLocation)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
}

func TestGetSQL(t *testing.T) {
	t.Parallel()
	inst := &Instance{}
	_, err := inst.GetSQL()
	if !errors.Is(err, errNilSQL) {
		t.Errorf("received %v, expected %v", err, errNilSQL)
	}

	databaseFullLocation := filepath.Join(DB.DataPath, "TestGetSQL")
	con, err := sql.Open("sqlite3", databaseFullLocation)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	err = inst.SetSQLiteConnection(con)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	_, err = inst.GetSQL()
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	inst = nil
	_, err = inst.GetSQL()
	if !errors.Is(err, ErrNilInstance) {
		t.Errorf("received %v, expected %v", err, ErrNilInstance)
	}
}
