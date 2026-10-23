package coinbase

import (
	"context"
)

// derivativesLogoutMethod closes the gateway's websocket connection
const derivativesLogoutMethod = "private/logout"

// Logout closes the gateway's websocket connection without cancelling its orders, even with cancel on disconnect
// enabled, and invalidates the session's tokens when invalidateToken is set; the gateway closes the connection without
// a response, and the websocket manager reconnects while the websocket is enabled. The gateway only accepts it over the
// websocket
func (w WebsocketAPI) Logout(ctx context.Context, invalidateToken bool) error {
	conn, err := w.connection(ctx, derivativesLogoutMethod)
	if err != nil {
		return err
	}
	// The gateway answers a logout by closing the connection, so no response is awaited
	return conn.SendJSONMessage(ctx, derivativesNonMatchingEPL, &JSONRPCRequest{
		JSONRPC: jsonRPCVersion,
		ID:      w.e.derivativesRequestID(),
		Method:  derivativesLogoutMethod,
		Params:  &DerivativesLogoutWireRequest{InvalidateToken: invalidateToken},
	})
}
