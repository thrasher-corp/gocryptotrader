package okx

import (
	"context"
	"errors"

	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

var errWebsocketOrderCredentialsMismatch = errors.New("order credentials do not match the authenticated websocket connection")

const maxWebsocketOrderCredentials = 32

// checkWebsocketOrderCredentials requires the order account to match this connection's confirmed login.
func (e *Exchange) checkWebsocketOrderCredentials(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	e.websocketOrderCredentialsMu.Lock()
	defer e.websocketOrderCredentialsMu.Unlock()
	loggedIn, ok := e.websocketOrderCredentials[conn]
	if !ok || loggedIn != *creds {
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
	if _, exists := e.websocketOrderCredentials[conn]; !exists && len(e.websocketOrderCredentials) >= maxWebsocketOrderCredentials {
		// Retired connections need not remain indefinitely. Missing identity
		// rejects websocket orders; the contract wrapper can fall back to REST.
		for previous := range e.websocketOrderCredentials {
			delete(e.websocketOrderCredentials, previous)
			break
		}
	}
	e.websocketOrderCredentials[conn] = *creds
}
