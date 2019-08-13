package engine

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	db "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	dbsqlite3 "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite"
	mg "github.com/thrasher-corp/gocryptotrader/database/migration"
	"github.com/thrasher-corp/gocryptotrader/database/repository/audit"
	auditPSQL "github.com/thrasher-corp/gocryptotrader/database/repository/audit/postgres"
	auditSQLite "github.com/thrasher-corp/gocryptotrader/database/repository/audit/sqlite"
	log "github.com/thrasher-corp/gocryptotrader/logger"
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

	log.Debugln(log.DatabaseMgr, "database manager starting...")

	a.shutdown = make(chan struct{})

	if Bot.Config.Database.Enabled {
		if Bot.Config.Database.Driver == "postgres" {
			dbConn, err = db.Connect()
			if err != nil {
				return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
			}

			dbConn.SQL.SetMaxOpenConns(2)
			dbConn.SQL.SetMaxIdleConns(1)
			dbConn.SQL.SetConnMaxLifetime(time.Hour)

			audit.Audit = auditPSQL.Audit()
		} else if Bot.Config.Database.Driver == "sqlite" {
			dbConn, err = dbsqlite3.Connect()

			if err != nil {
				return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
			}

			audit.Audit = auditSQLite.Audit()
		}
		dbConn.Connected = true

		mLogger := mg.MLogger{}
		migrations := mg.Migrator{
			Log: mLogger,
		}

		migrations.Conn = dbConn

		err := migrations.LoadMigrations()
		if err != nil {
			return err
		}

		err = migrations.RunMigration()
		if err != nil {
			return err
		}

		go a.run()
		return nil
	}

	return errors.New("database support disabled")
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

	t := time.NewTicker(time.Second * 30)
	a.running.Store(true)

	defer func() {
		t.Stop()
		a.running.Store(false)

		Bot.ServicesWG.Done()

		log.Debugln(log.DatabaseMgr, "database manager shutdown.")
	}()

	for {
		select {
		case <-a.shutdown:
			return
		case <-t.C:
			a.checkConnection()
		}
	}
}

func (a *databaseManager) checkConnection() {
	dbConn.Mu.Lock()
	err := dbConn.SQL.Ping()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "database connection error: %v", err)
		dbConn.Connected = false
		return
	}

	if !dbConn.Connected {
		log.Info(log.DatabaseMgr, "database connection reestablished")
		dbConn.Connected = true
	}

	dbConn.Mu.Lock()
}
