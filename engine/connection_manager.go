package engine

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/connchecker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// ConnectionManagerName is an exported subsystem name
const ConnectionManagerName = "internet_monitor"

var errConnectionCheckerIsNil = errors.New("connection checker is nil")

// connectionManager manages the connchecker
type connectionManager struct {
	started int32
	conn    *connchecker.Checker
	cfg     *config.ConnectionMonitorConfig
}

// IsRunning safely checks whether the subsystem is running
func (m *connectionManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// setupConnectionManager creates a connection manager
func setupConnectionManager(cfg *config.ConnectionMonitorConfig) (*connectionManager, error) {
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
	return &connectionManager{
		cfg: cfg,
	}, nil
}

// Start runs the subsystem
func (m *connectionManager) Start() error {
	if m == nil {
		return fmt.Errorf("connection manager %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("connection manager %w", ErrSubSystemAlreadyStarted)
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
func (m *connectionManager) Stop() error {
	if m == nil {
		return fmt.Errorf("connection manager: %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("connection manager: %w", ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()
	if m.conn == nil {
		return fmt.Errorf("connection manager: %w", errConnectionCheckerIsNil)
	}
	log.Debugln(log.ConnectionMgr, "Connection manager shutting down...")
	m.conn.Shutdown()
	log.Debugln(log.ConnectionMgr, "Connection manager stopped.")
	return nil
}

// IsOnline returns if the connection manager is online
func (m *connectionManager) IsOnline() bool {
	if m == nil {
		return false
	}
	if m.conn == nil {
		log.Warnln(log.ConnectionMgr, "Connection manager: IsOnline called but conn is nil")
		return false
	}

	return m.conn.IsConnected()
}
