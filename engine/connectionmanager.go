package engine

import (
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/connchecker"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystems"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// ConnectionManagerName is an exported subsystem name
const ConnectionManagerName = "internet_monitor"

// ConnectionManager manages the connchecker
type ConnectionManager struct {
	started int32
	conn    *connchecker.Checker
	cfg     *config.ConnectionMonitorConfig
}

// IsRunning safely checks whether the subsystem is running
func (m *ConnectionManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// SetupConnectionManager creates a connection manager
func SetupConnectionManager(cfg *config.ConnectionMonitorConfig) (*ConnectionManager, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if cfg.DNSList == nil {
		cfg.DNSList = connchecker.DefaultDNSList
	}
	if cfg.PublicDomainList == nil {
		cfg.PublicDomainList = connchecker.DefaultDomainList
	}
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = connchecker.DefaultCheckInterval
	}
	return &ConnectionManager{
		cfg: cfg,
	}, nil
}

// Start runs the subsystem
func (m *ConnectionManager) Start() error {
	if m == nil {
		return fmt.Errorf("connection manager %w", subsystems.ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("connection manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}

	log.Debugln(log.ConnectionMgr, "Connection manager starting...")
	var err error
	m.conn, err = connchecker.New(m.cfg.DNSList,
		m.cfg.PublicDomainList,
		m.cfg.CheckInterval)
	if err != nil {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
		return err
	}

	log.Debugln(log.ConnectionMgr, "Connection manager started.")
	return nil
}

// Stop stops the connection manager
func (m *ConnectionManager) Stop() error {
	if m == nil {
		return fmt.Errorf("connection manager %w", subsystems.ErrNilSubsystem)
	}
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
func (m *ConnectionManager) IsOnline() bool {
	if m == nil {
		return false
	}
	if m.conn == nil {
		log.Warnln(log.ConnectionMgr, "Connection manager: IsOnline called but conn is nil")
		return false
	}

	return m.conn.IsConnected()
}
