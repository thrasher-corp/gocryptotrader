package engine

import (
	"errors"
	"fmt"
	"github.com/thrasher-/gocryptotrader/database"
	"sync/atomic"
	"time"

	audit "github.com/thrasher-/gocryptotrader/database/repository/audit"

	db "github.com/thrasher-/gocryptotrader/database/drivers/postgresql"
	dbsqlite3 "github.com/thrasher-/gocryptotrader/database/drivers/sqlite"

	auditPSQL "github.com/thrasher-/gocryptotrader/database/repository/audit/postgres"
	auditSQLite "github.com/thrasher-/gocryptotrader/database/repository/audit/sqlite"

	log "github.com/thrasher-/gocryptotrader/logger"
)

var (
	dbConn *database.Database
)

type databaseManager struct {
	running  atomic.Value
	shutdown chan struct{}
}

func (a *databaseManager) Started() bool {
	return a.running.Load() == true
}

func (a *databaseManager) Start() (err error) {
	if a.Started() {
		return errors.New("database manager already started")
	}

	log.Debugln(log.AuditMgr, "database manager starting...")

	a.shutdown = make(chan struct{})

	if Bot.Config.Database.Driver == "postgres" {
		dbConn, err = db.Connect()
		if err != nil {
			return fmt.Errorf("Database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}

		dbConn.SQL.SetMaxOpenConns(2)
		dbConn.SQL.SetMaxIdleConns(1)
		dbConn.SQL.SetConnMaxLifetime(time.Hour)

		err = db.Setup()
		if err != nil {
			return err
		}

		audit.Audit = auditPSQL.Audit()
	} else if Bot.Config.Database.Driver == "sqlite" {
		dbConn, err = dbsqlite3.Connect()

		if err != nil {
			return fmt.Errorf("Database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}

		err = dbsqlite3.Setup()
		if err != nil {
			return err
		}
		audit.Audit = auditSQLite.Audit()
	}

	go a.run()

	return nil
}

func (a *databaseManager) Stop() error {
	if !a.Started() {
		return errors.New("database manager already stopped")
	}

	log.Debugln(log.DatabaseMgr, "database manager shutting down...")
	err := dbConn.SQL.Close()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Failed to close database: %v", err)
	}
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
