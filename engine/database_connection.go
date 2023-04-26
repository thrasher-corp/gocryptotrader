package engine

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/database"
	dbpsql "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	dbsqlite3 "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// DatabaseConnectionManagerName is an exported subsystem name
const DatabaseConnectionManagerName = "database"

// DatabaseConnectionManager holds the database connection and its status
type DatabaseConnectionManager struct {
	started  int32
	shutdown chan struct{}
	cfg      database.Config
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

// GetInstance returns a limited scoped database instance
func (m *DatabaseConnectionManager) GetInstance() database.IDatabase {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return nil
	}

	return m.dbConn
}

// SetupDatabaseConnectionManager creates a new database manager
func SetupDatabaseConnectionManager(cfg *database.Config) (*DatabaseConnectionManager, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	m := &DatabaseConnectionManager{
		shutdown: make(chan struct{}),
		cfg:      *cfg,
		dbConn:   database.DB,
	}
	if err := m.dbConn.SetConfig(cfg); err != nil {
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
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
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

	if m.cfg.Enabled {
		m.shutdown = make(chan struct{})
		switch m.cfg.Driver {
		case database.DBPostgreSQL:
			log.Debugf(log.DatabaseMgr,
				"Attempting to establish database connection to host %s/%s utilising %s driver\n",
				m.cfg.Host,
				m.cfg.Database,
				m.cfg.Driver)
			m.dbConn, err = dbpsql.Connect(&m.cfg)
		case database.DBSQLite,
			database.DBSQLite3:
			log.Debugf(log.DatabaseMgr,
				"Attempting to establish database connection to %s utilising %s driver\n",
				m.cfg.Database,
				m.cfg.Driver)
			m.dbConn, err = dbsqlite3.Connect(m.cfg.Database)
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

	return database.ErrDatabaseSupportDisabled
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
				log.Errorln(log.DatabaseMgr, "Database connection error:", err)
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
	if !m.cfg.Enabled {
		return database.ErrDatabaseSupportDisabled
	}
	if m.dbConn == nil {
		return database.ErrNoDatabaseProvided
	}

	if err := m.dbConn.Ping(); err != nil {
		m.dbConn.SetConnected(false)
		return err
	}

	if !m.dbConn.IsConnected() {
		log.Infoln(log.DatabaseMgr, "Database connection reestablished")
		m.dbConn.SetConnected(true)
	}
	return nil
}
