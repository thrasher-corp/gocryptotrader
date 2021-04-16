package base

import (
	"time"
)

// Base enforces standard variables across communication packages
type Base struct {
	Name           string
	Enabled        bool
	Verbose        bool
	Connected      bool
	ServiceStarted time.Time
}

// Event is a generalise event type
type Event struct {
	Type    string
	Message string
}

// CommsStatus stores the status of a comms relayer
type CommsStatus struct {
	Enabled   bool `json:"enabled"`
	Connected bool `json:"connected"`
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
	Service Started: ` + b.ServiceStarted.String()
}

func (b *Base) SetServiceStarted(t time.Time) {
	b.ServiceStarted = t
}
