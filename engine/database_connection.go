package engine

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
)

// DatabaseConnectionManagerName is an exported subsystem name
const DatabaseConnectionManagerName = "database"

var errDatabaseDisabled = errors.New("database support disabled")

// DatabaseConnectionManager holds the database connection and its status
type DatabaseConnectionManager struct {
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
func (m *DatabaseConnectionManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// SetupDatabaseConnectionManager creates a new database manager
func SetupDatabaseConnectionManager(cfg *database.Config) (*DatabaseConnectionManager, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	m := &DatabaseConnectionManager{
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

// IsConnected is an exported check to verify if the database is connected
func (m *DatabaseConnectionManager) IsConnected() bool {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return false
	}
	return m.dbConn.IsConnected()
}

// Start sets up the database connection manager to maintain a SQL connection
func (m *DatabaseConnectionManager) Start(wg *sync.WaitGroup) (err error) {
	if m == nil {
		return fmt.Errorf("%s %w", DatabaseConnectionManagerName, ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("database manager %w", ErrSubSystemAlreadyStarted)
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
			m.dbConn, err = dbpsql.Connect(m.dbConn.GetConfig())
		case database.DBSQLite,
			database.DBSQLite3:
			log.Debugf(log.DatabaseMgr,
				"Attempting to establish database connection to %s utilising %s driver\n",
				m.database,
				m.driver)
			m.dbConn, err = dbsqlite3.Connect(m.database)
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
func (m *DatabaseConnectionManager) Stop() error {
	if m == nil {
		return fmt.Errorf("%s %w", DatabaseConnectionManagerName, ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("%s %w", DatabaseConnectionManagerName, ErrSubSystemNotStarted)
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

func (m *DatabaseConnectionManager) run(wg *sync.WaitGroup) {
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

func (m *DatabaseConnectionManager) checkConnection() error {
	if m == nil {
		return fmt.Errorf("%s %w", DatabaseConnectionManagerName, ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("%s %w", DatabaseConnectionManagerName, ErrSubSystemNotStarted)
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
