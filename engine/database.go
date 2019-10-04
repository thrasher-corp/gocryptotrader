package engine

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	dbpsql "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	dbsqlite3 "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/sqlboiler/boil"
)

var (
	dbConn *database.Db
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

	log.Debugln(log.DatabaseMgr, "Database manager starting...")

	a.shutdown = make(chan struct{})

	if Bot.Config.Database.Enabled {
		if Bot.Config.Database.Driver == database.DBPostgreSQL {
			log.Debugf(log.DatabaseMgr, "Attempting to establish database connection to host %s/%s utilising %s driver\n",
				Bot.Config.Database.Host, Bot.Config.Database.Database, Bot.Config.Database.Driver)
			dbConn, err = dbpsql.Connect()
		} else if Bot.Config.Database.Driver == database.DBSQLite || Bot.Config.Database.Driver == database.DBSQLite3 {
			log.Debugf(log.DatabaseMgr, "Attempting to establish database connection to %s utilising %s driver\n",
				Bot.Config.Database.Database, Bot.Config.Database.Driver)
			dbConn, err = dbsqlite3.Connect()
		}
		if err != nil {
			return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}
		dbConn.Connected = true

		DBLogger := database.Logger{}
		if Bot.Config.Database.Verbose {
			boil.DebugMode = true
			boil.DebugWriter = DBLogger
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

	log.Debugln(log.DatabaseMgr, "Database manager shutting down...")

	err := dbConn.SQL.Close()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Failed to close database: %v", err)
	}

	close(a.shutdown)
	return nil
}

func (a *databaseManager) run() {
	log.Debugln(log.DatabaseMgr, "Database manager started.")
	Bot.ServicesWG.Add(1)

	t := time.NewTicker(time.Second * 2)

	a.running.Store(true)

	defer func() {
		t.Stop()
		a.running.Store(false)

		Bot.ServicesWG.Done()

		log.Debugln(log.DatabaseMgr, "Database manager shutdown.")
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
	defer dbConn.Mu.Unlock()

	err := dbConn.SQL.Ping()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Database connection error: %v\n", err)
		dbConn.Connected = false
		return
	}

	if !dbConn.Connected {
		log.Info(log.DatabaseMgr, "Database connection reestablished")
		dbConn.Connected = true
	}
}
