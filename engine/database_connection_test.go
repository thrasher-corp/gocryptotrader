package engine

import (
	"errors"
	"log"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
)

//nolint:gocritic // Only used as a testing helper function in this package
func CreateDatabase(t *testing.T) {
	t.Helper()
	// fun workarounds to globals ruining testing
	database.DB.DataPath = t.TempDir()

	t.Cleanup(func() {
		if database.DB.IsConnected() {
			err := database.DB.CloseConnection()
			if err != nil {
				log.Fatal(err)
			}
		}
	})
}

func TestSetupDatabaseConnectionManager(t *testing.T) {
	_, err := SetupDatabaseConnectionManager(nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	m, err := SetupDatabaseConnectionManager(&database.Config{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestStartSQLite(t *testing.T) {
	CreateDatabase(t)
	m, err := SetupDatabaseConnectionManager(&database.Config{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	var wg sync.WaitGroup
	err = m.Start(&wg)
	if !errors.Is(err, database.ErrDatabaseSupportDisabled) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrDatabaseSupportDisabled)
	}
	m, err = SetupDatabaseConnectionManager(&database.Config{Enabled: true})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start(&wg)
	if !errors.Is(err, database.ErrNoDatabaseProvided) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrNoDatabaseProvided)
	}
	m.cfg = database.Config{Driver: database.DBSQLite}
	err = m.Start(&wg)
	if !errors.Is(err, database.ErrDatabaseSupportDisabled) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrDatabaseSupportDisabled)
	}
	_, err = SetupDatabaseConnectionManager(&database.Config{
		Enabled: true,
		Driver:  database.DBSQLite,
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Database: "test.db",
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

// This test does not care for a successful connection
func TestStartPostgres(t *testing.T) {
	m, err := SetupDatabaseConnectionManager(&database.Config{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	var wg sync.WaitGroup
	err = m.Start(&wg)
	if !errors.Is(err, database.ErrDatabaseSupportDisabled) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrDatabaseSupportDisabled)
	}
	m.cfg.Enabled = true
	err = m.Start(&wg)
	if !errors.Is(err, database.ErrNoDatabaseProvided) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrNoDatabaseProvided)
	}
	m.cfg.Driver = database.DBPostgreSQL
	err = m.Start(&wg)
	if !errors.Is(err, database.ErrFailedToConnect) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrFailedToConnect)
	}
}

func TestDatabaseConnectionManagerIsRunning(t *testing.T) {
	CreateDatabase(t)
	m, err := SetupDatabaseConnectionManager(&database.Config{
		Enabled: true,
		Driver:  database.DBSQLite,
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Database: "test.db",
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m.IsRunning() {
		t.Error("expected false")
	}
	var wg sync.WaitGroup
	err = m.Start(&wg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if !m.IsRunning() {
		t.Error("expected true")
	}
	m = nil
	if m.IsRunning() {
		t.Error("expected false")
	}
}

func TestDatabaseConnectionManagerStop(t *testing.T) {
	CreateDatabase(t)
	m, err := SetupDatabaseConnectionManager(&database.Config{
		Enabled: true,
		Driver:  database.DBSQLite,
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Database: "test.db",
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	var wg sync.WaitGroup
	err = m.Start(&wg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	m = nil
	err = m.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestCheckConnection(t *testing.T) {
	CreateDatabase(t)
	var m *DatabaseConnectionManager
	err := m.checkConnection()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
	m, err = SetupDatabaseConnectionManager(&database.Config{
		Enabled: true,
		Driver:  database.DBSQLite,
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Database: "test.db",
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.checkConnection()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	var wg sync.WaitGroup
	err = m.Start(&wg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.checkConnection()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.checkConnection()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	err = m.Start(&wg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.checkConnection()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	m.dbConn.SetConnected(false)
	err = m.checkConnection()
	if !errors.Is(err, database.ErrDatabaseNotConnected) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrDatabaseNotConnected)
	}

	err = m.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestGetInstance(t *testing.T) {
	CreateDatabase(t)
	m, err := SetupDatabaseConnectionManager(&database.Config{
		Enabled: true,
		Driver:  database.DBSQLite,
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Database: "test.db",
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	db := m.GetInstance()
	if db != nil {
		t.Error("expected nil")
	}
	var wg sync.WaitGroup
	err = m.Start(&wg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	db = m.GetInstance()
	if db == nil {
		t.Error("expected not nil")
	}

	m = nil
	db = m.GetInstance()
	if db != nil {
		t.Error("expected nil")
	}
}
