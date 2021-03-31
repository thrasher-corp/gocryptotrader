package communicationmanager

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/config"

	"github.com/thrasher-corp/gocryptotrader/communications"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
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

func (m *Manager) Start(cfg *config.CommunicationsConfig) (err error) {
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("communications manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}
	defer func() {
		if err != nil {
			atomic.CompareAndSwapInt32(&m.started, 1, 0)
		}
	}()

	log.Debugln(log.CommunicationMgr, "Communications manager starting...")
	m.comms, err = communications.NewComm(cfg)
	if err != nil {
		return err
	}

	m.shutdown = make(chan struct{})
	m.relayMsg = make(chan base.Event)
	go m.run()
	log.Debugln(log.CommunicationMgr, "Communications manager started.")
	return nil
}

func (m *Manager) GetStatus() (map[string]base.CommsStatus, error) {
	if !m.Started() {
		return nil, errors.New("communications manager not started")
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
	log.Debugln(log.CommunicationMgr, "Communications manager shutting down...")
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
		log.Debugln(log.CommunicationMgr, "Communications manager shutdown.")
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
