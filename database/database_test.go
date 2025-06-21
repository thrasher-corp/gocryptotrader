package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestSetConfig(t *testing.T) {
	t.Parallel()
	inst := &Instance{}
	err := inst.SetConfig(&Config{Verbose: true})
	assert.NoError(t, err)

	err = inst.SetConfig(nil)
	assert.ErrorIs(t, err, ErrNilConfig)

	inst = nil
	err = inst.SetConfig(&Config{})
	assert.ErrorIs(t, err, ErrNilInstance)
}

func TestSetSQLiteConnection(t *testing.T) {
	t.Parallel()
	inst := &Instance{}
	err := inst.SetSQLiteConnection(nil)
	assert.ErrorIs(t, err, errNilSQL)

	err = inst.SetSQLiteConnection(&sql.DB{})
	assert.NoError(t, err)

	inst = nil
	err = inst.SetSQLiteConnection(nil)
	assert.ErrorIs(t, err, ErrNilInstance)
}

func TestSetPostgresConnection(t *testing.T) {
	// there is nothing actually requiring a postgres connection specifically
	// so this is testing the checks and the ability to set values
	// however, such settings would be bad for a sqlite connection irl
	t.Parallel()
	inst := &Instance{}
	databaseFullLocation := filepath.Join(DB.DataPath, "TestSetPostgresConnection")
	con, err := sql.Open("sqlite3", databaseFullLocation)
	assert.NoError(t, err)

	err = inst.SetPostgresConnection(con)
	assert.NoError(t, err)

	err = con.Close()
	assert.NoError(t, err)

	err = os.Remove(databaseFullLocation)
	assert.NoError(t, err)
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
	assert.NoError(t, err)

	err = inst.SetSQLiteConnection(con)
	assert.NoError(t, err)

	err = inst.CloseConnection()
	assert.NoError(t, err)
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
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	err = inst.SetSQLiteConnection(con)
	assert.NoError(t, err)

	inst.SetConnected(true)
	err = inst.Ping()
	assert.NoError(t, err)

	inst.SQL = nil
	err = inst.Ping()
	assert.ErrorIs(t, err, errNilSQL)

	inst.SetConnected(false)
	err = inst.Ping()
	assert.ErrorIs(t, err, ErrDatabaseNotConnected)

	inst = nil
	err = inst.Ping()
	assert.ErrorIs(t, err, ErrNilInstance)

	err = con.Close()
	assert.NoError(t, err)

	err = os.Remove(databaseFullLocation)
	assert.NoError(t, err)
}

func TestGetSQL(t *testing.T) {
	t.Parallel()
	inst := &Instance{}
	_, err := inst.GetSQL()
	assert.ErrorIs(t, err, errNilSQL)

	databaseFullLocation := filepath.Join(DB.DataPath, "TestGetSQL")
	con, err := sql.Open("sqlite3", databaseFullLocation)
	assert.NoError(t, err)

	err = inst.SetSQLiteConnection(con)
	assert.NoError(t, err)

	_, err = inst.GetSQL()
	assert.NoError(t, err)

	inst = nil
	_, err = inst.GetSQL()
	assert.ErrorIs(t, err, ErrNilInstance)
}
