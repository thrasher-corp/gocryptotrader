package connectionmanager

import (
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/connchecker"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
)

// Manager manages the connchecker
type Manager struct {
	started int32
	conn    *connchecker.Checker
}

// Started returns if the connection manager has started
func (m *Manager) Started() bool {
	return atomic.LoadInt32(&m.started) == 1
}

// Start starts an instance of the connection manager
func (m *Manager) Start(conf *config.ConnectionMonitorConfig) error {
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("connection manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}

	log.Debugln(log.ConnectionMgr, "Connection manager starting...")
	var err error
	m.conn, err = connchecker.New(conf.DNSList,
		conf.PublicDomainList,
		conf.CheckInterval)
	if err != nil {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
		return err
	}

	log.Debugln(log.ConnectionMgr, "Connection manager started.")
	return nil
}

// Stop stops the connection manager
func (m *Manager) Stop() error {
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("connection manager %w", subsystems.ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()

	log.Debugln(log.ConnectionMgr, "Connection manager shutting down...")
	m.conn.Shutdown()
	log.Debugln(log.ConnectionMgr, "Connection manager stopped.")
	return nil
}

// IsOnline returns if the connection manager is online
func (m *Manager) IsOnline() bool {
	if m.conn == nil {
		log.Warnln(log.ConnectionMgr, "Connection manager: IsOnline called but conn is nil")
		return false
	}

	return m.conn.IsConnected()
}
