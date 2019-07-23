package engine

import (
	"errors"
	"sync/atomic"

	log "github.com/thrasher-/gocryptotrader/logger"
)

type auditManager struct {
	running  atomic.Value
	shutdown chan struct{}
}

func (a *auditManager) Started() bool {
	return a.running.Load() == true
}

func (a *auditManager) Start() error {
	if a.Started() {
		return errors.New("audit manager already started")
	}

	log.Debugln(log.AuditMgr, "Audit manager starting...")

	a.shutdown = make(chan struct{})
	go a.run()

	return nil
}

func (a *auditManager) Stop() error {
	if !a.Started() {
		return errors.New("audit manager already stopped")
	}

	log.Debugln(log.AuditMgr, "Audit manager shutting down...")
	close(a.shutdown)

	return nil
}

func (a *auditManager) run() {
	log.Debugln(log.AuditMgr, "Audit manager started.")
	Bot.ServicesWG.Add(1)

	a.running.Store(true)

	defer func() {
		a.running.Store(false)

		Bot.ServicesWG.Done()

		log.Debugln(log.AuditMgr, "Audit manager shutdown.")
	}()

	for {
		select {
		case <-a.shutdown:
			return
		}
	}
}
