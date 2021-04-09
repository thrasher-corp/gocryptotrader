package databaseconnection

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
)

func TestMain(m *testing.M) {
	// fun workarounds to globals ruining testing
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	database.DB.DataPath = tmp
	os.Exit(m.Run())
}

func TestSetup(t *testing.T) {
	t.Parallel()
	_, err := Setup(nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	m, err := Setup(&database.Config{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestStartSQLite(t *testing.T) {
	t.Parallel()
	m, err := Setup(&database.Config{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	var wg sync.WaitGroup
	err = m.Start(&wg)
	if !errors.Is(err, errDatabaseDisabled) {
		t.Errorf("error '%v', expected '%v'", err, errDatabaseDisabled)
	}
	m.enabled = true
	err = m.Start(&wg)
	if !errors.Is(err, database.ErrNoDatabaseProvided) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrNoDatabaseProvided)
	}
	m.driver = database.DBSQLite
	err = m.Start(&wg)
	if !errors.Is(err, database.ErrFailedToConnect) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrFailedToConnect)
	}
	m, err = Setup(&database.Config{
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
	t.Parallel()
	m, err := Setup(&database.Config{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	var wg sync.WaitGroup
	err = m.Start(&wg)
	if !errors.Is(err, errDatabaseDisabled) {
		t.Errorf("error '%v', expected '%v'", err, errDatabaseDisabled)
	}
	m.enabled = true
	err = m.Start(&wg)
	if !errors.Is(err, database.ErrNoDatabaseProvided) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrNoDatabaseProvided)
	}
	m.driver = database.DBPostgreSQL
	err = m.Start(&wg)
	if !errors.Is(err, database.ErrFailedToConnect) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrFailedToConnect)
	}
}

func TestIsRunning(t *testing.T) {
	t.Parallel()
	m, err := Setup(&database.Config{
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

func TestStop(t *testing.T) {
	t.Parallel()
	m, err := Setup(&database.Config{
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
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
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
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}
}

func TestCheckConnection(t *testing.T) {
	t.Parallel()
	var m *Manager
	err := m.checkConnection()
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}
	m, err = Setup(&database.Config{
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
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
	}
	m.started = 1
	err = m.checkConnection()
	if !errors.Is(err, database.ErrNoDatabaseProvided) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrNoDatabaseProvided)
	}
	m.enabled = false
	err = m.checkConnection()
	if !errors.Is(err, database.ErrDatabaseSupportDisabled) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrDatabaseSupportDisabled)
	}
	m.started = 0
	m.enabled = true
	var wg sync.WaitGroup
	err = m.Start(&wg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.checkConnection()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	m.dbConn.Connected = false
	err = m.checkConnection()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}
