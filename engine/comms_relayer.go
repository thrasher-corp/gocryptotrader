package engine

import (
	"errors"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/communications"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// commsManager starts the NTP manager
type commsManager struct {
	started  int32
	stopped  int32
	shutdown chan struct{}
	relayMsg chan base.Event
	comms    *communications.Communications
}

func (c *commsManager) Started() bool {
	return atomic.LoadInt32(&c.started) == 1
}

func (c *commsManager) Start() (err error) {
	if atomic.AddInt32(&c.started, 1) != 1 {
		return errors.New("communications manager already started")
	}

	defer func() {
		if err != nil {
			atomic.CompareAndSwapInt32(&c.started, 1, 0)
		}
	}()

	log.Debugln(log.CommunicationMgr, "Communications manager starting...")
	commsCfg := Bot.Config.GetCommunicationsConfig()
	c.comms, err = communications.NewComm(&commsCfg)
	if err != nil {
		return err
	}

	c.shutdown = make(chan struct{})
	c.relayMsg = make(chan base.Event)
	go c.run()
	log.Debugln(log.CommunicationMgr, "Communications manager started.")
	return nil
}

func (c *commsManager) GetStatus() (map[string]base.CommsStatus, error) {
	if !c.Started() {
		return nil, errors.New("communications manager not started")
	}
	return c.comms.GetStatus(), nil
}

func (c *commsManager) Stop() error {
	if atomic.LoadInt32(&c.started) == 0 {
		return errors.New("communications manager not started")
	}

	if atomic.AddInt32(&c.stopped, 1) != 1 {
		return errors.New("communications manager is already stopped")
	}

	close(c.shutdown)
	log.Debugln(log.CommunicationMgr, "Communications manager shutting down...")
	return nil
}

func (c *commsManager) PushEvent(evt base.Event) {
	if !c.Started() {
		return
	}
	select {
	case c.relayMsg <- evt:
	default:
		log.Errorf(log.CommunicationMgr, "Failed to send, no receiver when pushing event [%v]", evt)
	}
}

func (c *commsManager) run() {
	defer func() {
		// TO-DO shutdown comms connections for connected services (Slack etc)
		atomic.CompareAndSwapInt32(&c.stopped, 1, 0)
		atomic.CompareAndSwapInt32(&c.started, 1, 0)
		log.Debugln(log.CommunicationMgr, "Communications manager shutdown.")
	}()

	for {
		select {
		case msg := <-c.relayMsg:
			c.comms.PushEvent(msg)
		case <-c.shutdown:
			return
		}
	}
}
