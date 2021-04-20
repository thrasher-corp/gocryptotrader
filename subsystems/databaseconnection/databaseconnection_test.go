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
	database.DB.DataPath = tmp
	code := m.Run()
	err = os.RemoveAll(tmp)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(code)
}

func TestSetup(t *testing.T) {
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
	m, err := Setup(&database.Config{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	var wg sync.WaitGroup
	err = m.Start(&wg)
	if !errors.Is(err, errDatabaseDisabled) {
		t.Errorf("error '%v', expected '%v'", err, errDatabaseDisabled)
	}
	m, err = Setup(&database.Config{Enabled: true})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
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
	var wg sync.WaitGroup
	err = m.Start(&wg)
	err = m.checkConnection()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.checkConnection()
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
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
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}
