package communicationmanager

import (
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/communications"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
)

// Manager starts the commuications manager
type Manager struct {
	started  int32
	shutdown chan struct{}
	relayMsg chan base.Event
	comms    *communications.Communications
}

func (m *Manager) Started() bool {
	return atomic.LoadInt32(&m.started) == 1
}

func (m *Manager) Setup(cfg *config.CommunicationsConfig) (*Manager, error) {
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return nil, fmt.Errorf("communications manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}

	manager := &Manager{
		shutdown: make(chan struct{}),
		relayMsg: make(chan base.Event),
	}
	var err error
	log.Debugf(log.Global, "Communications manager %s", subsystems.MsgSubSystemStarting)
	manager.comms, err = communications.NewComm(cfg)
	if err != nil {
		return nil, err
	}
	go manager.run()
	log.Debugf(log.Global, "Communications manager %s", subsystems.MsgSubSystemStarted)
	return manager, nil
}

func (m *Manager) GetStatus() (map[string]base.CommsStatus, error) {
	if !m.Started() {
		return nil, fmt.Errorf("communications manager %w", subsystems.ErrSubSystemNotStarted)
	}
	return m.comms.GetStatus(), nil
}

func (m *Manager) Stop() error {
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

func (m *Manager) PushEvent(evt base.Event) {
	if !m.Started() {
		return
	}
	select {
	case m.relayMsg <- evt:
	default:
		log.Errorf(log.CommunicationMgr, "Failed to send, no receiver when pushing event [%v]", evt)
	}
}

func (m *Manager) run() {
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
