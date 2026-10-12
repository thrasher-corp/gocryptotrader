package okx

import (
	"context"
	"errors"

	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

var errWebsocketOrderCredentialsMismatch = errors.New("order credentials do not match the authenticated websocket connection")

// checkWebsocketOrderCredentials requires the order account to match this connection's confirmed login.
func (e *Exchange) checkWebsocketOrderCredentials(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	e.websocketOrderCredentialsMu.Lock()
	defer e.websocketOrderCredentialsMu.Unlock()
	loggedIn, ok := e.websocketOrderCredentials[conn]
	if !ok || loggedIn.Key != creds.Key || loggedIn.Secret != creds.Secret || loggedIn.ClientID != creds.ClientID {
		return errWebsocketOrderCredentialsMismatch
	}
	return nil
}

// setWebsocketOrderCredentials snapshots a confirmed login; nil invalidates it before re-authentication.
func (e *Exchange) setWebsocketOrderCredentials(conn websocket.Connection, creds *accounts.Credentials) {
	e.websocketOrderCredentialsMu.Lock()
	defer e.websocketOrderCredentialsMu.Unlock()
	if creds == nil {
		delete(e.websocketOrderCredentials, conn)
		return
	}
	if e.websocketOrderCredentials == nil {
		e.websocketOrderCredentials = make(map[websocket.Connection]accounts.Credentials)
	}
	// Retired sockets must not displace the confirmed identity of a live connection.
	for previous := range e.websocketOrderCredentials {
		if previous != conn {
			if connection, ok := previous.(interface{ IsConnected() bool }); ok && !connection.IsConnected() {
				delete(e.websocketOrderCredentials, previous)
			}
		}
	}
	e.websocketOrderCredentials[conn] = *creds
}

// canUseWebsocketForAccount selects a socket only when its confirmed login matches the requesting account.
func (e *Exchange) canUseWebsocketForAccount(ctx context.Context, connection string) bool {
	// The manager can hold its lock while reconnecting; REST-only requests must not wait on it.
	if !e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		return false
	}
	conn, err := e.Websocket.GetConnection(connection)
	return err == nil && e.checkWebsocketOrderCredentials(ctx, conn) == nil
}
