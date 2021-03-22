package engine

import (
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/connchecker"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// connectionManager manages the connchecker
type connectionManager struct {
	started int32
	conn    *connchecker.Checker
}

// Started returns if the connection manager has started
func (c *connectionManager) Started() bool {
	return atomic.LoadInt32(&c.started) == 1
}

// Start starts an instance of the connection manager
func (c *connectionManager) Start(conf *config.ConnectionMonitorConfig) error {
	if !atomic.CompareAndSwapInt32(&c.started, 0, 1) {
		return fmt.Errorf("connection manager %w", subsystem.ErrSubSystemAlreadyStarted)
	}

	log.Debugln(log.ConnectionMgr, "Connection manager starting...")
	var err error
	c.conn, err = connchecker.New(conf.DNSList,
		conf.PublicDomainList,
		conf.CheckInterval)
	if err != nil {
		atomic.CompareAndSwapInt32(&c.started, 1, 0)
		return err
	}

	log.Debugln(log.ConnectionMgr, "Connection manager started.")
	return nil
}

// Stop stops the connection manager
func (c *connectionManager) Stop() error {
	if atomic.LoadInt32(&c.started) == 0 {
		return fmt.Errorf("connection manager %w", subsystem.ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&c.started, 1, 0)
	}()

	log.Debugln(log.ConnectionMgr, "Connection manager shutting down...")
	c.conn.Shutdown()
	log.Debugln(log.ConnectionMgr, "Connection manager stopped.")
	return nil
}

// IsOnline returns if the connection manager is online
func (c *connectionManager) IsOnline() bool {
	if c.conn == nil {
		log.Warnln(log.ConnectionMgr, "Connection manager: IsOnline called but conn is nil")
		return false
	}

	return c.conn.IsConnected()
}
