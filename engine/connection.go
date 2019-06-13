package engine

import (
	"errors"
	"sync/atomic"

	"github.com/thrasher-/gocryptotrader/connchecker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// connectionManager manages the connchecker
type connectionManager struct {
	started int32
	stopped int32
	conn    *connchecker.Checker
}

func (c *connectionManager) Started() bool {
	return atomic.LoadInt32(&c.started) == 1
}

func (c *connectionManager) Start() error {
	if atomic.AddInt32(&c.started, 1) != 1 {
		return errors.New("connection manager already started")
	}

	log.Debugln("Connection manager starting...")
	var err error
	c.conn, err = connchecker.New(Bot.Config.ConnectionMonitor.DNSList,
		Bot.Config.ConnectionMonitor.PublicDomainList,
		Bot.Config.ConnectionMonitor.CheckInterval)
	if err != nil {
		atomic.CompareAndSwapInt32(&c.started, 1, 0)
		return err
	}

	log.Debugln("Connection manager started.")
	return nil
}

func (c *connectionManager) Stop() error {
	if atomic.LoadInt32(&c.started) == 0 {
		return errors.New("connection manager not started")
	}

	if atomic.AddInt32(&c.stopped, 1) != 1 {
		return errors.New("connection manager is already stopped")
	}

	log.Debugln("Connection manager shutting down...")
	c.conn.Shutdown()
	atomic.CompareAndSwapInt32(&c.stopped, 1, 0)
	atomic.CompareAndSwapInt32(&c.started, 1, 0)
	log.Debugln("Connection manager stopped.")
	return nil
}

func (c *connectionManager) IsOnline() bool {
	if c.conn == nil {
		log.Warnf("Connection manager: IsOnline called but conn is nil")
		return false
	}

	return c.conn.IsConnected()
}
