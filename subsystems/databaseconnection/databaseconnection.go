package databaseconnection

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	dbpsql "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	dbsqlite3 "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
)

// Name is an exported subsystem name
const Name = "database"

var (
	errNilConfig        = errors.New("received nil database config")
	errDatabaseDisabled = errors.New("database support disabled")
)

// Manager holds the database connection and its status
type Manager struct {
	started  int32
	shutdown chan struct{}
	enabled  bool
	verbose  bool
	host     string
	username string
	password string
	database string
	driver   string
	wg       sync.WaitGroup
	dbConn   *database.Instance
}

// IsRunning safely checks whether the subsystem is running
func (m *Manager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Setup creates a new database manager
func Setup(cfg *database.Config) (*Manager, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	m := &Manager{
		shutdown: make(chan struct{}),
		enabled:  cfg.Enabled,
		verbose:  cfg.Verbose,
		host:     cfg.Host,
		username: cfg.Username,
		password: cfg.Password,
		database: cfg.Database,
		driver:   cfg.Driver,
		dbConn:   database.DB,
	}
	err := m.dbConn.SetConfig(cfg)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Start sets up the database connection manager to maintain a SQL connection
func (m *Manager) Start(wg *sync.WaitGroup) (err error) {
	if m == nil {
		return fmt.Errorf("%s %w", Name, subsystems.ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("database manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}
	defer func() {
		if err != nil {
			atomic.CompareAndSwapInt32(&m.started, 1, 0)
		}
	}()

	log.Debugln(log.DatabaseMgr, "Database manager starting...")

	if m.enabled {
		m.shutdown = make(chan struct{})
		switch m.driver {
		case database.DBPostgreSQL:
			log.Debugf(log.DatabaseMgr,
				"Attempting to establish database connection to host %s/%s utilising %s driver\n",
				m.host,
				m.database,
				m.driver)
			m.dbConn, err = dbpsql.Connect()
		case database.DBSQLite,
			database.DBSQLite3:
			log.Debugf(log.DatabaseMgr,
				"Attempting to establish database connection to %s utilising %s driver\n",
				m.database,
				m.driver)
			m.dbConn, err = dbsqlite3.Connect()
		default:
			return database.ErrNoDatabaseProvided
		}
		if err != nil {
			return fmt.Errorf("%w: %v Some features that utilise a database will be unavailable", database.ErrFailedToConnect, err)
		}
		m.dbConn.SetConnected(true)
		wg.Add(1)
		m.wg.Add(1)
		go m.run(wg)
		return nil
	}

	return errDatabaseDisabled
}

// Stop stops the database manager and closes the connection
// Stop attempts to shutdown the subsystem
func (m *Manager) Stop() error {
	if m == nil {
		return fmt.Errorf("%s %w", Name, subsystems.ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("%s %w", Name, subsystems.ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()

	err := m.dbConn.CloseConnection()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Failed to close database: %v", err)
	}

	close(m.shutdown)
	m.wg.Wait()
	return nil
}

func (m *Manager) run(wg *sync.WaitGroup) {
	log.Debugln(log.DatabaseMgr, "Database manager started.")
	t := time.NewTicker(time.Second * 2)

	defer func() {
		t.Stop()
		m.wg.Done()
		wg.Done()
		log.Debugln(log.DatabaseMgr, "Database manager shutdown.")
	}()

	for {
		select {
		case <-m.shutdown:
			return
		case <-t.C:
			err := m.checkConnection()
			if err != nil {
				log.Error(log.DatabaseMgr, "Database connection error:", err)
			}
		}
	}
}

func (m *Manager) checkConnection() error {
	if m == nil {
		return fmt.Errorf("%s %w", Name, subsystems.ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("%s %w", Name, subsystems.ErrSubSystemNotStarted)
	}
	if !m.enabled {
		return database.ErrDatabaseSupportDisabled
	}
	if m.dbConn == nil {
		return database.ErrNoDatabaseProvided
	}

	err := m.dbConn.Ping()
	if err != nil {
		m.dbConn.SetConnected(false)
		return err
	}

	if !m.dbConn.IsConnected() {
		log.Info(log.DatabaseMgr, "Database connection reestablished")
		m.dbConn.SetConnected(true)
	}
	return nil
}
