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
	"github.com/thrasher-corp/sqlboiler/boil"
)

// Manager holds the database connection and its status
type Manager struct {
	started  int32
	shutdown chan struct{}
	dbConn   *database.Instance
}

// Started returns whether the database connection manager is running
func (m *Manager) Started() bool {
	return atomic.LoadInt32(&m.started) == 1
}

// Start sets up the database connection manager to maintain a SQL connection
func (m *Manager) Start(cfg *database.Config, wg *sync.WaitGroup) (err error) {
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("database manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}

	defer func() {
		if err != nil {
			atomic.CompareAndSwapInt32(&m.started, 1, 0)
		}
	}()

	log.Debugln(log.DatabaseMgr, "Database manager starting...")

	m.shutdown = make(chan struct{})

	if cfg.Enabled {
		if cfg.Driver == database.DBPostgreSQL {
			log.Debugf(log.DatabaseMgr,
				"Attempting to establish database connection to host %s/%s utilising %s driver\n",
				cfg.Host,
				cfg.Database,
				cfg.Driver)
			m.dbConn, err = dbpsql.Connect()
		} else if cfg.Driver == database.DBSQLite ||
			cfg.Driver == database.DBSQLite3 {
			log.Debugf(log.DatabaseMgr,
				"Attempting to establish database connection to %s utilising %s driver\n",
				cfg.Database,
				cfg.Driver)
			m.dbConn, err = dbsqlite3.Connect()
		}
		if err != nil {
			return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}
		m.dbConn.Connected = true

		DBLogger := database.Logger{}
		if cfg.Verbose {
			boil.DebugMode = true
			boil.DebugWriter = DBLogger
		}
		wg.Add(1)
		go m.run(wg)
		return nil
	}

	return errors.New("database support disabled")
}

// Stop stops the database manager and closes the connection
func (m *Manager) Stop() error {
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("database manager %w", subsystems.ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()

	err := m.dbConn.SQL.Close()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Failed to close database: %v", err)
	}

	close(m.shutdown)
	return nil
}

func (m *Manager) run(wg *sync.WaitGroup) {
	log.Debugln(log.DatabaseMgr, "Database manager started.")
	t := time.NewTicker(time.Second * 2)

	defer func() {
		t.Stop()
		wg.Done()
		log.Debugln(log.DatabaseMgr, "Database manager shutdown.")
	}()

	for {
		select {
		case <-m.shutdown:
			return
		case <-t.C:
			go m.checkConnection()
		}
	}
}

func (m *Manager) checkConnection() {
	m.dbConn.Mu.Lock()
	defer m.dbConn.Mu.Unlock()

	err := m.dbConn.SQL.Ping()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Database connection error: %v\n", err)
		m.dbConn.Connected = false
		return
	}

	if !m.dbConn.Connected {
		log.Info(log.DatabaseMgr, "Database connection reestablished")
		m.dbConn.Connected = true
	}
}
