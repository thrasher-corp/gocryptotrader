package base

import (
	"errors"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// IComm is the main interface array across the communication packages
type IComm []ICommunicate

// ICommunicate enforces standard functions across communication packages
type ICommunicate interface {
	Setup(config *config.CommunicationsConfig)
	Connect() error
	PushEvent(Event) error
	IsEnabled() bool
	IsConnected() bool
	GetName() string
}

// Setup sets up communication variables and intiates a connection to the
// communication mediums
func (c IComm) Setup() {
	ServiceStarted = time.Now()
	for i := range c {
		if c[i].IsEnabled() && !c[i].IsConnected() {
			err := c[i].Connect()
			if err != nil {
				log.Errorf("Communications: %s failed to connect. Err: %s", c[i].GetName(), err)
			}
		}
	}
}

// PushEvent pushes triggered events to all enabled communication links
func (c IComm) PushEvent(event Event) {
	for i := range c {
		if c[i].IsEnabled() && c[i].IsConnected() {
			err := c[i].PushEvent(event)
			if err != nil {
				log.Errorf("Communications error - PushEvent() in package %s with %v. Err %s",
					c[i].GetName(), event, err)
			}
		}
	}
}

// GetEnabledCommunicationMediums prints out enabled and connected communication
// packages
// (#debug output only)
func (c IComm) GetEnabledCommunicationMediums() error {
	var count int
	for i := range c {
		if c[i].IsEnabled() && c[i].IsConnected() {
			log.Debugf("Communications: Medium %s is enabled.", c[i].GetName())
			count++
		}
	}
	if count == 0 {
		return errors.New("no communication mediums are enabled")
	}
	return nil
}
