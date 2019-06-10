package base

import (
	"time"
)

// global vars contain staged update data that will be sent to the communication
// mediums
var (
	ServiceStarted time.Time
)

// Base enforces standard variables across communication packages
type Base struct {
	Name      string
	Enabled   bool
	Verbose   bool
	Connected bool
}

// Event is a generalise event type
type Event struct {
	Type    string
	Message string
}

// IsEnabled returns if the comms package has been enabled in the configuration
func (b *Base) IsEnabled() bool {
	return b.Enabled
}

// IsConnected returns if the package is connected to a server and/or ready to
// send
func (b *Base) IsConnected() bool {
	return b.Connected
}

// GetName returns a package name
func (b *Base) GetName() string {
	return b.Name
}

// GetStatus returns status data
func (b *Base) GetStatus() string {
	return `
	GoCryptoTrader Service: Online
	Service Started: ` + ServiceStarted.String()
}
