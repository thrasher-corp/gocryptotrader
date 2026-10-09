package websocket

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// SetSubscriptionsNotRequired configures the manager to connect without
// generating default subscriptions.
//
// This exported helper exists for cross-package test harnesses only. It is a
// pre-connect test-mode mutation used by websocket unit-test helpers that need
// request-response connectivity without normal stream bootstraps. Tests that
// later need the manager's default subscription generation should use a fresh
// exchange/manager instance.
func (m *Manager) SetSubscriptionsNotRequired() {
	m.m.Lock()
	defer m.m.Unlock()

	if !m.useMultiConnectionManagement {
		m.GenerateSubs = func() (subscription.List, error) { return subscription.List{}, nil }
		return
	}

	for _, ws := range m.connectionManager {
		if ws.setup == nil {
			log.Warnf(log.WebsocketMgr, "%s websocket: missing connection setup while disabling required subscriptions; creating empty setup", m.exchangeName)
			ws.setup = &ConnectionSetup{}
		}
		ws.setup.SubscriptionsNotRequired = true
	}
}

func (m *Manager) testWebsocketByFilter(messageFilter any) (*websocket, error) {
	if err := common.NilGuard(m); err != nil {
		return nil, err
	}
	if !m.useMultiConnectionManagement {
		return nil, fmt.Errorf("%s: multi connection management not enabled %w please use exported Conn and AuthConn fields", m.exchangeName, errCannotObtainOutboundConnection)
	}
	for _, ws := range m.snapshotConnectionManager() {
		if ws.setup != nil && ws.setup.MessageFilter == messageFilter {
			return ws, nil
		}
	}
	return nil, fmt.Errorf("%s: %w associated with message filter: '%v'", m.exchangeName, ErrRequestRouteNotFound, messageFilter)
}

// CreateTestConnection returns an unmanaged connection built from the setup
// matching the supplied message filter.
//
// This exported helper exists for cross-package test harnesses only. It lets
// exchange-package tests reuse a manager-configured connection type without
// establishing a live websocket session.
func (m *Manager) CreateTestConnection(messageFilter any) (Connection, error) {
	ws, err := m.testWebsocketByFilter(messageFilter)
	if err != nil {
		return nil, err
	}
	if ws.setup == nil {
		return nil, fmt.Errorf("%w: ConnectionSetup", common.ErrNilPointer)
	}
	return m.createConnectionFromSetup(ws.setup), nil
}

// CreateUnmanagedTestConnection returns an isolated connection for handler tests.
// It deliberately has its own matcher and subscription store and is registered
// only for connection-specific subscription lookup; it is not added to the
// manager's connect or shutdown lifecycle.
func (m *Manager) CreateUnmanagedTestConnection(u string) Connection {
	conn := m.createConnectionFromSetup(&ConnectionSetup{URL: u, ResponseMaxLimit: time.Second})
	conn.Match = NewMatch()
	m.connectionManagerMu.Lock()
	m.connections[conn] = &websocket{subscriptions: conn.subscriptions}
	m.connectionManagerMu.Unlock()
	return conn
}

// TrackTestConnection registers a test connection against the managed
// websocket matching the supplied message filter.
//
// This exported helper exists for cross-package test harnesses only. It is
// intended for tests that wrap a manager-created connection with custom
// send/receive behaviour while still needing multi-connection bookkeeping and
// GetConnection routing without starting a live connection lifecycle.
func (m *Manager) TrackTestConnection(messageFilter any, conn Connection) error {
	if err := common.NilGuard(conn); err != nil {
		return err
	}
	ws, err := m.testWebsocketByFilter(messageFilter)
	if err != nil {
		return err
	}
	m.trackConnection(conn, ws)
	m.setState(connectedState)
	return nil
}

// SetAllConnectionURLs configures every managed websocket connection to use the
// same URL.
//
// This exported helper exists for cross-package test harnesses only. It is a
// pre-connect test-mode mutation used by websocket unit-test helpers to
// redirect all websocket traffic through a single mock server before
// connecting. Calling this after Connect has started returns an error.
func (m *Manager) SetAllConnectionURLs(u string) error {
	if err := common.NilGuard(m); err != nil {
		return err
	}
	if err := checkWebsocketURL(u); err != nil {
		return err
	}

	m.m.Lock()
	defer m.m.Unlock()

	if m.IsConnecting() {
		return fmt.Errorf("%v %w: SetAllConnectionURLs must be called before Connect", m.exchangeName, errAlreadyReconnecting)
	}
	if m.IsConnected() {
		return fmt.Errorf("%v %w: SetAllConnectionURLs must be called before Connect", m.exchangeName, errAlreadyConnected)
	}

	if !m.useMultiConnectionManagement {
		m.runningURL = u
		m.runningURLAuth = u
		if m.Conn != nil {
			m.Conn.SetURL(u)
		}
		if m.AuthConn != nil {
			m.AuthConn.SetURL(u)
		}
		return nil
	}

	for _, ws := range m.connectionManager {
		if ws.setup == nil {
			log.Warnf(log.WebsocketMgr, "%s websocket: missing connection setup while updating connection URLs; creating empty setup", m.exchangeName)
			ws.setup = &ConnectionSetup{}
		}
		ws.setup.URL = u
	}
	return nil
}
