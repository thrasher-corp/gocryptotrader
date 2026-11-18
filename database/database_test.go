package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var errCloseFailed = errors.New("close failed")

// closeErrorConnector backs a pool whose Close fails
type closeErrorConnector struct{}

func (closeErrorConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, driver.ErrBadConn
}

func (closeErrorConnector) Driver() driver.Driver { return nil }

func (closeErrorConnector) Close() error { return errCloseFailed }

func openMemoryDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err, "sql.Open must not error")
	t.Cleanup(func() { assert.NoError(t, db.Close(), "Close should not error") })
	return db
}

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

	err = con.Close()
	assert.NoError(t, err)

	inst = nil
	_, err = inst.GetSQL()
	assert.ErrorIs(t, err, ErrNilInstance)
}

func TestConnectionReplacement(t *testing.T) {
	t.Parallel()
	for name, set := range map[string]func(*Instance, *sql.DB) error{
		"sqlite":   (*Instance).SetSQLiteConnection,
		"postgres": (*Instance).SetPostgresConnection,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			inst := &Instance{}
			first := openMemoryDatabase(t)
			require.NoError(t, set(inst, first), "setting a connection must not error")

			second := openMemoryDatabase(t)
			require.NoError(t, set(inst, second), "replacing the connection must not error")
			assert.Same(t, second, inst.SQL, "the replacement should be installed")
			assert.ErrorContains(t, first.PingContext(t.Context()), "database is closed", "the replaced connection should be closed")
		})
	}
}

func TestReplaceSQL(t *testing.T) {
	// Not parallel: the log hook and log config are process-wide, and cleanup removes the hook and disables logging
	type logEntry struct{ header, message string }
	var mu sync.Mutex
	var logged []logEntry
	require.NoError(t, log.SetGlobalLogConfig(log.GenDefaultSettings()), "SetGlobalLogConfig must not error")
	require.NoError(t, log.SetupGlobalLogger(t.Name(), false), "SetupGlobalLogger must not error")
	log.SetCustomLogHook(func(header, _ string, a ...any) bool {
		mu.Lock()
		logged = append(logged, logEntry{header, fmt.Sprint(a...)})
		mu.Unlock()
		return true
	})
	t.Cleanup(func() {
		log.SetCustomLogHook(nil)
		assert.NoError(t, log.SetGlobalLogConfig(&log.Config{}), "SetGlobalLogConfig should not error")
	})

	inst := &Instance{}
	first := openMemoryDatabase(t)
	inst.replaceSQL(first)
	require.Same(t, first, inst.SQL, "replaceSQL must install the first connection")
	inst.replaceSQL(first)
	require.Same(t, first, inst.SQL, "replaceSQL must keep the same connection installed")
	assert.NoError(t, first.PingContext(t.Context()), "replaceSQL should leave the same connection open")

	failsToClose := sql.OpenDB(closeErrorConnector{})
	inst.replaceSQL(failsToClose)
	require.Same(t, failsToClose, inst.SQL, "replaceSQL must install the replacement")
	assert.ErrorContains(t, first.PingContext(t.Context()), "database is closed", "replaceSQL should close the connection it replaces")

	last := openMemoryDatabase(t)
	inst.replaceSQL(last)
	assert.Same(t, last, inst.SQL, "replaceSQL should install the replacement when closing the replaced connection fails")
	assert.ErrorContains(t, failsToClose.PingContext(t.Context()), "database is closed", "replaceSQL should close the replaced connection when its Close fails")
	assert.NoError(t, last.PingContext(t.Context()), "replaceSQL should leave the replacement open")

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, logged, 1, "replaceSQL must log only the failed close")
	assert.Equal(t, "[ERROR]", logged[0].header, "replaceSQL should log the failed close as an error")
	assert.Contains(t, logged[0].message, errCloseFailed.Error(), "replaceSQL should log the close error")
}
