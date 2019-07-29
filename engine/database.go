package engine

import (
	"errors"
	"fmt"
	audit "github.com/thrasher-/gocryptotrader/db/repository/audit"
	"sync/atomic"
	"time"

	db "github.com/thrasher-/gocryptotrader/db/drivers/postgresql"

	auditRepo "github.com/thrasher-/gocryptotrader/db/repository/audit/postgres"

	log "github.com/thrasher-/gocryptotrader/logger"
)

type databaseManager struct {
	running  atomic.Value
	shutdown chan struct{}
}

func (a *databaseManager) Started() bool {
	return a.running.Load() == true
}

func (a *databaseManager) Start() error {
	if a.Started() {
		return errors.New("database manager already started")
	}

	log.Debugln(log.AuditMgr, "database manager starting...")

	a.shutdown = make(chan struct{})

	//connStr := drivers.ConnectionDetails{
	//	Host:     "127.0.0.1",
	//	Port:     5432,
	//	Database: "gct-audit",
	//	Username: "gct",
	//	Password: "test1234",
	//}

	dbConn, err := db.ConnectPSQL()
	if err != nil {
		return fmt.Errorf("Database failed to connect: %v Some features that utilise a database will be unavailable", err)
	}

	dbConn.SQL.SetMaxOpenConns(2)
	dbConn.SQL.SetMaxIdleConns(1)
	dbConn.SQL.SetConnMaxLifetime(time.Hour)

	audit.Audit = auditRepo.NewPSQLAudit()

	go a.run()

	return nil
}

func (a *databaseManager) Stop() error {
	if !a.Started() {
		return errors.New("database manager already stopped")
	}

	log.Debugln(log.DatabaseMgr, "database manager shutting down...")

	close(a.shutdown)
	return nil
}

func (a *databaseManager) run() {
	log.Debugln(log.DatabaseMgr, "database manager started.")
	Bot.ServicesWG.Add(1)

	a.running.Store(true)

	defer func() {
		a.running.Store(false)

		Bot.ServicesWG.Done()

		log.Debugln(log.DatabaseMgr, "database manager shutdown.")
	}()

	for {
		select {
		case <-a.shutdown:
			return
		}
	}
}
