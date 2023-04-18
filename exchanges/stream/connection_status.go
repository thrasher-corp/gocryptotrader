package stream

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var (
	// ErrProtocolNotEnabled is returned when a protocol is not enabled
	ErrProtocolNotEnabled = errors.New("protocol not enabled")
	// ErrProtocolNotInitialized is returned when a protocol is not initialized
	ErrProtocolNotInitialized = errors.New("protocol not initialized")
	// ErrProtocolNotConnecting is returned when a protocol is not connecting
	ErrProtocolNotConnecting = errors.New("protocol not connecting")
	// ErrProtocolAlreadyConnected is returned when a protocol is already connected
	ErrProtocolAlreadyConnected = errors.New("protocol already connected")
	// ErrProtocolNotConnected is returned when a protocol is not connected
	ErrProtocolNotConnected = errors.New("protocol not connected")
	// ErrProtocolAlreadyNotConnected is returned when a protocol is already not connected
	ErrProtocolAlreadyNotConnected = errors.New("protocol already not connected")
	// ErrProtocolAlreadyConnecting is returned when a protocol is already connecting
	ErrProtocolAlreadyConnecting = errors.New("protocol already connecting")
	// ErrProtocolAlreadyEnabled is returned when a protocol is already enabled
	ErrProtocolAlreadyEnabled = errors.New("protocol already enabled")
	// ErrProtocolAlreadyDisabled is returned when a protocol is already disabled
	ErrProtocolAlreadyDisabled = errors.New("protocol already disabled")
	// ErrProtocolAlreadyInitialized is returned when a protocol is already initialized
	ErrProtocolAlreadyInitialized = errors.New("protocol already initialized")
	// ErrProtocolAlreadySubscribed is returned when a protocol is already subscribed
	ErrProtocolAlreadySubscribed = errors.New("protocol already subscribed")
	// ErrProtocolNotSubscribed is returned when a protocol is not subscribed
	ErrProtocolNotSubscribed = errors.New("protocol not subscribed")
	// ErrProtocolAlreadyUnsubscribed is returned when a protocol is already unsubscribed
	ErrProtocolAlreadyUnsubscribed = errors.New("protocol already unsubscribed")
	// ErrProtocolAlreadyRunning is returned when a protocol is already running
	ErrProtocolAlreadyRunning = errors.New("protocol already running")
	// ErrProtocolNotRunning is returned when a protocol is not running
	ErrProtocolNotRunning = errors.New("protocol not running")
	// ErrProtocolAlreadyStopped is returned when a protocol is already stopped
	ErrProtocolAlreadyStopped = errors.New("protocol already stopped")
	// ErrProtocolNotStopped is returned when a protocol is not stopped
	ErrProtocolNotStopped = errors.New("protocol not stopped")
	// ErrProtocolAlreadyDisconnected is returned when a protocol is already disconnected
	ErrProtocolAlreadyDisconnected = errors.New("protocol already disconnected")
	// ErrProtocolAlreadyNotDisconnected is returned when a protocol is already not disconnected
	ErrProtocolAlreadyNotDisconnected = errors.New("protocol already not disconnected")
	// ErrProtocolAlreadyShutdown is returned when a protocol is already shutdown
	ErrProtocolAlreadyShutdown = errors.New("protocol already shutdown")

	// ErrMonitorAlreadyRunning is returned when a monitor is already running
	ErrMonitorAlreadyRunning = errors.New("monitor already running")
	// ErrMonitorAlreadyStopped is returned when a monitor is already stopped
	ErrMonitorAlreadyStopped = errors.New("monitor already stopped")

	// ErrProtocolAlreadyShuttingDown is returned when a protocol is shutting down
	ErrProtocolShuttingDown = errors.New("protocol shutting down")

	// ErrProtocolNotShuttingDown is returned when a protocol is not shutting down
	ErrProtocolNotShuttingDown = errors.New("protocol not shutting down")
)

// ConnectionStatus stores connection status information
type ConnectionStatus struct {
	Enabled                  bool
	Init                     bool
	Connected                bool
	Connecting               bool
	ShuttingDown             bool
	ConnectionMonitorRunning bool
	TrafficMonitorRunning    bool
	DataMonitorRunning       bool

	// wg tracks datamonitor, connectionmonitor and trafficmonitor
	wg  sync.WaitGroup
	mtx sync.RWMutex
}

// attemptConnection changes state to connecting and returns an error if it is
// unable to do so.
func (c *ConnectionStatus) attemptConnection() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.Enabled {
		return ErrProtocolNotEnabled
	}

	if c.Init {
		return ErrProtocolAlreadyInitialized
	}

	if c.Connected {
		return ErrProtocolAlreadyConnected
	}

	if c.Connecting {
		return ErrProtocolAlreadyConnecting
	}

	if c.ShuttingDown {
		return ErrProtocolShuttingDown
	}

	c.Connecting = true
	return nil
}

// attemptDisconnect changes state to not connected and returns an error if it
// is unable to do so.
func (c *ConnectionStatus) attempShutdown() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.Enabled {
		return ErrProtocolNotEnabled
	}

	if !c.Init {
		return ErrProtocolNotInitialized
	}

	if !c.Connected {
		return ErrProtocolNotConnected
	}

	if c.ShuttingDown {
		return ErrProtocolShuttingDown
	}
	c.ShuttingDown = true
	return nil
}

// connectionShutdown changes state to not connected and returns an error if it
// is unable to do so.
func (c *ConnectionStatus) connectionShutdown() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.Enabled {
		return ErrProtocolNotEnabled
	}

	if !c.Init {
		return ErrProtocolNotInitialized
	}

	if !c.Connected {
		return ErrProtocolAlreadyNotConnected
	}

	if !c.ShuttingDown {
		return ErrProtocolNotShuttingDown
	}

	c.Connected = false
	c.ShuttingDown = false
	return nil
}

// connectionEstablished changes state to connected and returns an error if it
// is unable to do so.
func (c *ConnectionStatus) connectionEstablished() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.Enabled {
		return ErrProtocolNotEnabled
	}

	if !c.Init {
		return ErrProtocolNotInitialized
	}

	if c.Connected {
		return ErrProtocolAlreadyConnected
	}

	if !c.Connecting {
		return ErrProtocolNotConnecting
	}

	if c.ShuttingDown {
		return ErrProtocolShuttingDown
	}

	c.Connected = true
	c.Connecting = false
	return nil
}

// connectionFailed changes state to not connected and returns an error if it is
// unable to do so.
func (c *ConnectionStatus) connectionFailed(err error) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.Enabled {
		return common.AppendError(err, ErrProtocolNotEnabled)
	}

	if !c.Init {
		return common.AppendError(err, ErrProtocolNotInitialized)
	}

	if c.Connected {
		return common.AppendError(err, ErrProtocolAlreadyConnected)
	}

	if !c.Connecting {
		return common.AppendError(err, ErrProtocolNotConnecting)
	}

	c.Connecting = false
	return err
}

// setConnectedStatus sets the connection status
func (c *ConnectionStatus) setConnectedStatus(isConnected bool) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.Enabled && isConnected {
		return ErrProtocolNotEnabled
	}

	if !c.Init && isConnected {
		return ErrProtocolNotInitialized
	}

	if !c.Connecting && isConnected {
		return ErrProtocolNotConnecting
	}

	if c.Connected && isConnected {
		return ErrProtocolAlreadyConnected
	}

	if !c.Connected && !isConnected {
		return ErrProtocolAlreadyNotConnected
	}

	if c.ShuttingDown {
		return ErrProtocolShuttingDown
	}

	c.Connected = isConnected
	return nil
}

// setConnectingStatus sets the connecting status
func (c *ConnectionStatus) setConnectingStatus(isConnecting bool) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.Enabled && isConnecting {
		return ErrProtocolNotEnabled
	}

	if !c.Init && isConnecting {
		return ErrProtocolNotInitialized
	}

	if c.Connecting && isConnecting {
		return ErrProtocolAlreadyConnecting
	}

	if c.Connected && isConnecting {
		return ErrProtocolAlreadyConnected
	}

	if !c.Connected && !isConnecting {
		return ErrProtocolAlreadyNotConnected
	}

	if c.ShuttingDown {
		return ErrProtocolShuttingDown
	}

	c.Connecting = isConnecting
	return nil
}

// setInit sets the initial connection status
func (c *ConnectionStatus) setInit(isInitialStart bool) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.Enabled && isInitialStart {
		return ErrProtocolNotEnabled
	}

	if c.Init && isInitialStart {
		return ErrProtocolAlreadyConnected
	}

	if c.Connected && isInitialStart {
		return ErrProtocolAlreadyConnected
	}

	if !c.Connected && !isInitialStart {
		return ErrProtocolAlreadyNotConnected
	}

	if !c.Connecting && isInitialStart {
		return ErrProtocolNotConnecting
	}

	if c.Connecting && !isInitialStart {
		return ErrProtocolNotConnecting
	}
	if !c.Init && !isInitialStart {
		return ErrProtocolNotInitialized
	}
	if c.Init && isInitialStart {
		return ErrProtocolAlreadyConnected
	}

	if c.ShuttingDown {
		return ErrProtocolShuttingDown
	}

	c.Init = isInitialStart
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

// setTrafficMonitorRunning sets the traffic monitor status
func (c *ConnectionStatus) setTrafficMonitorRunning(isRunning bool) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if !c.Enabled && isRunning {
		return ErrProtocolNotEnabled
	}

	if !c.Init && isRunning {
		return ErrProtocolNotInitialized
	}

	if c.TrafficMonitorRunning && isRunning {
		return ErrMonitorAlreadyRunning
	}

	if !c.TrafficMonitorRunning && !isRunning {
		return ErrMonitorAlreadyStopped
	}

	c.TrafficMonitorRunning = isRunning
	return nil
}

func (c *ConnectionStatus) setConnectionMonitorRunning(isRunning bool) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if !c.Enabled && isRunning {
		return ErrProtocolNotEnabled
	}

	if !c.Init && isRunning {
		return ErrProtocolNotInitialized
	}

	if c.ConnectionMonitorRunning && isRunning {
		return ErrMonitorAlreadyRunning
	}

	if !c.ConnectionMonitorRunning && !isRunning {
		return ErrMonitorAlreadyStopped
	}

	c.ConnectionMonitorRunning = isRunning
	return nil
}

func (c *ConnectionStatus) setDataMonitorRunning(isRunning bool) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if !c.Enabled && isRunning {
		return ErrProtocolNotEnabled
	}

	if !c.Init && isRunning {
		return ErrProtocolNotInitialized
	}

	if c.DataMonitorRunning && isRunning {
		return ErrMonitorAlreadyRunning
	}

	if !c.DataMonitorRunning && !isRunning {
		return ErrMonitorAlreadyStopped
	}

	c.DataMonitorRunning = isRunning
	return nil
}

func (c *ConnectionStatus) checkAndSetMonitorRunning() (alreadyRunning bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.ConnectionMonitorRunning {
		return true
	}
	c.ConnectionMonitorRunning = true
	return false
}

// IsConnected returns status of connection
func (c *ConnectionStatus) IsConnected() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.Connected
}

// IsConnecting returns status of connecting
func (c *ConnectionStatus) IsConnecting() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.Connecting
}

// IsEnabled returns status of enabled
func (c *ConnectionStatus) IsEnabled() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.Enabled
}

// IsInit returns status of init
func (c *ConnectionStatus) IsInit() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.Init
}

// IsTrafficMonitorRunning returns status of the traffic monitor
func (c *ConnectionStatus) IsTrafficMonitorRunning() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.TrafficMonitorRunning
}

// IsConnectionMonitorRunning returns status of connection monitor
func (c *ConnectionStatus) IsConnectionMonitorRunning() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.ConnectionMonitorRunning
}

// IsDataMonitorRunning returns status of data monitor
func (c *ConnectionStatus) IsDataMonitorRunning() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.DataMonitorRunning
}

// IsDisconnected returns if connection is disconnected
func (c *ConnectionStatus) IsDisconnected() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return !c.Connecting && !c.Connected
}
