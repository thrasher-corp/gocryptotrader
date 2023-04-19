package stream

import (
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrProtocolNotEnabled is returned when a protocol is not enabled
	ErrProtocolNotEnabled = errors.New("protocol not enabled")
	// ErrProtocolAlreadyConnected is returned when a protocol is already connected
	ErrProtocolAlreadyConnected = errors.New("protocol already connected")
	// ErrProtocolNotConnected is returned when a protocol is not connected
	ErrProtocolNotConnected = errors.New("protocol not connected")
	// ErrProtocolAlreadyEnabled is returned when a protocol is already enabled
	ErrProtocolAlreadyEnabled = errors.New("protocol already enabled")
	// ErrProtocolAlreadyDisabled is returned when a protocol is already disabled
	ErrProtocolAlreadyDisabled = errors.New("protocol already disabled")
)

// ConnectionStatus stores connection status information
type ConnectionStatus struct {
	Enabled      bool
	Connected    bool
	Connector    func() error
	ExchangeName string
	mtx          sync.RWMutex
}

// Connect attempts to establish a connection to an exchange's protocol stream
// service. If the connection is successful, it will start the connection
// monitor, which will monitor the connection and attempt to reconnect if the
// connection is lost. It will also start the processing monitor, which will
// monitor the prcoessing of data.
func (c *ConnectionStatus) Connect() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.Enabled {
		return fmt.Errorf("%s error establishing stream connection: %w", c.ExchangeName, ErrProtocolNotEnabled)
	}

	if c.Connected {
		return fmt.Errorf("%s error establishing stream connection: %w", c.ExchangeName, ErrProtocolAlreadyConnected)
	}

	if c.Connector == nil {
		return fmt.Errorf("%s error establishing stream connection: %w", c.ExchangeName, errors.New("connector not set"))
	}

	err := c.Connector()
	if err != nil {
		return fmt.Errorf("%s error establishing stream connection: %w", c.ExchangeName, err)
	}

	c.Connected = true
	return nil
}

// Shutdown changes state to disconnected. Ensure that the connection is
// disconnected before calling this function as there is not awareness currently
// on connection status.
func (c *ConnectionStatus) Shutdown() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.Enabled {
		return fmt.Errorf("%s error shutting down stream connection: %w", c.ExchangeName, ErrProtocolNotEnabled)
	}

	if !c.Connected {
		return fmt.Errorf("%s error shutting down stream connection: %w", c.ExchangeName, ErrProtocolNotConnected)
	}

	c.Connected = false
	return nil
}

// setEnabled sets the enabled status
func (c *ConnectionStatus) setEnabled(isEnabled bool) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.Enabled && isEnabled {
		return ErrProtocolAlreadyEnabled
	}
	if !c.Enabled && !isEnabled {
		return ErrProtocolAlreadyDisabled
	}
	c.Enabled = isEnabled
	return nil
}

// IsConnected returns status of connection
func (c *ConnectionStatus) IsConnected() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.Connected
}

// IsEnabled returns status of enabled
func (c *ConnectionStatus) IsEnabled() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.Enabled
}
