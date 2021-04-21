package connectionmanager

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/connchecker"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
)

// Name is an exported subsystem name
const Name = "internet_monitor"

// Manager manages the connchecker
type Manager struct {
	started int32
	conn    *connchecker.Checker
	cfg     *config.ConnectionMonitorConfig
}

var errNilConfig = errors.New("nil config")

// IsRunning safely checks whether the subsystem is running
func (m *Manager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Setup creates a database connection manager
func Setup(cfg *config.ConnectionMonitorConfig) (*Manager, error) {
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
	return &Manager{
		cfg: cfg,
	}, nil
}

// Start runs the subsystem
func (m *Manager) Start() error {
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
// Stop attempts to shutdown the subsystem
func (m *Manager) Stop() error {
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
func (m *Manager) IsOnline() bool {
	if m == nil {
		return false
	}
	if m.conn == nil {
		log.Warnln(log.ConnectionMgr, "Connection manager: IsOnline called but conn is nil")
		return false
	}

	return m.conn.IsConnected()
}
