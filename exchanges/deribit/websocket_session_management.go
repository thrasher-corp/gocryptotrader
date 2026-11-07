package deribit

import (
	"context"
	"fmt"
	"time"
)

// SetHeartbeat makes Deribit send heartbeats and test requests over the websocket at an interval of whole seconds, at
// least 10, closing the connection when a test request goes unanswered; Deribit only accepts it over the websocket
func (w WebsocketAPI) SetHeartbeat(ctx context.Context, interval time.Duration) error {
	if interval < minimumHeartbeatInterval {
		return fmt.Errorf("%w: %s", errHeartbeatIntervalTooShort, interval)
	}
	if interval%time.Second != 0 {
		return fmt.Errorf("%w: %s", errHeartbeatIntervalNotWholeSeconds, interval)
	}
	return w.e.sendRequestMatchOKResponse(ctx, true, "public/set_heartbeat", &SetHeartbeatRequest{Interval: uint64(interval.Seconds())})
}

// DisableHeartbeat stops Deribit sending heartbeats over the websocket; Deribit only accepts it over the websocket
func (w WebsocketAPI) DisableHeartbeat(ctx context.Context) error {
	return w.e.sendRequestMatchOKResponse(ctx, true, "public/disable_heartbeat", nil)
}

// EnableCancelOnDisconnect makes Deribit cancel the orders a connection placed once it closes, other than by
// WebsocketAPI.Logout; the connection scope, which an empty scope defaults to, applies to this connection, and the
// account scope to every connection opened afterwards. Deribit only accepts it over the websocket
func (w WebsocketAPI) EnableCancelOnDisconnect(ctx context.Context, scope string) error {
	if err := validateCancelOnDisconnectScope(scope); err != nil {
		return err
	}
	return w.e.sendRequestMatchOKResponse(ctx, true, "private/enable_cancel_on_disconnect", &CancelOnDisconnectRequest{Scope: scope})
}

// DisableCancelOnDisconnect stops Deribit cancelling the orders a connection placed once it closes, for the connection
// or, with the account scope, for every connection opened afterwards; Deribit only accepts it over the websocket
func (w WebsocketAPI) DisableCancelOnDisconnect(ctx context.Context, scope string) error {
	if err := validateCancelOnDisconnectScope(scope); err != nil {
		return err
	}
	return w.e.sendRequestMatchOKResponse(ctx, true, "private/disable_cancel_on_disconnect", &CancelOnDisconnectRequest{Scope: scope})
}

// GetCancelOnDisconnect returns whether the connection, which an empty scope defaults to, or the account cancels a
// connection's orders once it closes
func (w WebsocketAPI) GetCancelOnDisconnect(ctx context.Context, scope string) (*CancelOnDisconnectResponse, error) {
	return w.e.getCancelOnDisconnect(ctx, scope, true)
}
