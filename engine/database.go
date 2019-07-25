package engine

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/thrasher-/gocryptotrader/db/drivers"
	db "github.com/thrasher-/gocryptotrader/db/drivers/postgresql"

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

	connStr := drivers.ConnectionDetails{
		Host:     "127.0.0.1",
		Port:     5432,
		Database: "gct-audit",
		Username: "gct",
		Password: "test1234",
	}

	dbConn, err := db.ConnectPSQL(connStr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dbConn.SQL.SetMaxOpenConns(2)
	dbConn.SQL.SetMaxIdleConns(1)
	dbConn.SQL.SetConnMaxLifetime(time.Hour)

	go a.run()

	return nil
}

func (a *databaseManager) Stop() error {
	if !a.Started() {
		return errors.New("database manager already stopped")
	}

	log.Debugln(log.AuditMgr, "database manager shutting down...")

	close(a.shutdown)
	return nil
}

func (a *databaseManager) run() {
	log.Debugln(log.AuditMgr, "database manager started.")
	Bot.ServicesWG.Add(1)

	a.running.Store(true)

	defer func() {
		a.running.Store(false)

		Bot.ServicesWG.Done()

		log.Debugln(log.AuditMgr, "database manager shutdown.")
	}()

	for {
		select {
		case <-a.shutdown:
			return
		}
	}
}
