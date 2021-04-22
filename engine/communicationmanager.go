package engine

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/communications"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystems"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Name is an exported subsystem name
const Name = "communications"

// CommunicationManager ensures operations of communications
type CommunicationManager struct {
	started  int32
	shutdown chan struct{}
	relayMsg chan base.Event
	comms    *communications.Communications
}

var errNilConfig = errors.New("received nil communications config")

// SetupCommunicationManager creates a communications manager
func SetupCommunicationManager(cfg *config.CommunicationsConfig) (*CommunicationManager, error) {
	if cfg == nil {
		return nil, errNilConfig
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
		return fmt.Errorf("communications manager server %w", subsystems.ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("communications manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}
	log.Debugf(log.CommunicationMgr, "Communications manager %s", subsystems.MsgSubSystemStarting)
	m.shutdown = make(chan struct{})
	go m.run()
	return nil
}

// GetStatus returns the status of communications
func (m *CommunicationManager) GetStatus() (map[string]base.CommsStatus, error) {
	if !m.IsRunning() {
		return nil, fmt.Errorf("communications manager %w", subsystems.ErrSubSystemNotStarted)
	}
	return m.comms.GetStatus(), nil
}

// Stop attempts to shutdown the subsystem
func (m *CommunicationManager) Stop() error {
	if m == nil {
		return fmt.Errorf("communications manager server %w", subsystems.ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("communications manager %w", subsystems.ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()
	close(m.shutdown)
	log.Debugf(log.CommunicationMgr, "Communications manager %s", subsystems.MsgSubSystemShuttingDown)
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
	log.Debugf(log.Global, "Communications manager %s", subsystems.MsgSubSystemStarted)
	defer func() {
		// TO-DO shutdown comms connections for connected services (Slack etc)
		log.Debugf(log.CommunicationMgr, "Communications manager %s", subsystems.MsgSubSystemShutdown)
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
