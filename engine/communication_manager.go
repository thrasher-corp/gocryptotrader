package engine

import (
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/communications"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// CommunicationsManagerName is an exported subsystem name
const CommunicationsManagerName = "communications"

// CommunicationManager ensures operations of communications
type CommunicationManager struct {
	started  int32
	shutdown chan struct{}
	relayMsg chan base.Event
	comms    *communications.Communications
}

// SetupCommunicationManager creates a communications manager
func SetupCommunicationManager(cfg *base.CommunicationsConfig) (*CommunicationManager, error) {
	if cfg == nil {
		return nil, subsystem.ErrNilConfig
	}
	manager := &CommunicationManager{
		shutdown: make(chan struct{}),
		relayMsg: make(chan base.Event),
	}
	var err error
	manager.comms, err = communications.NewComm(cfg)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

// IsRunning safely checks whether the subsystem is running
func (m *CommunicationManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Start runs the subsystem
func (m *CommunicationManager) Start() error {
	if m == nil {
		return fmt.Errorf("communications manager server %w", subsystem.ErrNil)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("communications manager %w", subsystem.ErrAlreadyStarted)
	}
	log.Debugf(log.CommunicationMgr, "Communications manager %s", subsystem.MsgStarting)
	m.shutdown = make(chan struct{})
	go m.run()
	return nil
}

// GetStatus returns the status of communications
func (m *CommunicationManager) GetStatus() (map[string]base.CommsStatus, error) {
	if !m.IsRunning() {
		return nil, fmt.Errorf("communications manager %w", subsystem.ErrNotStarted)
	}
	return m.comms.GetStatus(), nil
}

// Stop attempts to shutdown the subsystem
func (m *CommunicationManager) Stop() error {
	if m == nil {
		return fmt.Errorf("communications manager server %w", subsystem.ErrNil)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("communications manager %w", subsystem.ErrNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()
	close(m.shutdown)
	log.Debugf(log.CommunicationMgr, "Communications manager %s", subsystem.MsgShuttingDown)
	return nil
}

// PushEvent pushes an event to the communications relay
func (m *CommunicationManager) PushEvent(evt base.Event) {
	if !m.IsRunning() {
		return
	}
	select {
	case m.relayMsg <- evt:
	default:
		log.Errorf(log.CommunicationMgr, "Failed to send, no receiver when pushing event [%v]", evt)
	}
}

// run takes awaiting messages and pushes them to be handled by communications
func (m *CommunicationManager) run() {
	log.Debugf(log.Global, "Communications manager %s", subsystem.MsgStarted)
	defer func() {
		// TO-DO shutdown comms connections for connected services (Slack etc)
		log.Debugf(log.CommunicationMgr, "Communications manager %s", subsystem.MsgShutdown)
	}()

	for {
		select {
		case msg := <-m.relayMsg:
			m.comms.PushEvent(msg)
		case <-m.shutdown:
			return
		}
	}
}
