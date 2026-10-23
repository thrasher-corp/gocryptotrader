package coinbase

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// derivativesMinimumHeartbeatInterval is the shortest heartbeat interval the gateway accepts
const derivativesMinimumHeartbeatInterval = 10 * time.Second

var (
	errDerivativesHeartbeatIntervalTooShort = errors.New("gateway heartbeat interval must be at least 10 seconds")
	errDerivativesHeartbeatIntervalFraction = errors.New("gateway heartbeat interval must be whole seconds")
)

// DisableCancelOnDisconnect stops the gateway cancelling the orders a connection placed once it closes, for this
// connection or, with the account scope, for every connection opened afterwards; the gateway documents it as websocket
// only
func (w WebsocketAPI) DisableCancelOnDisconnect(ctx context.Context, scope string) error {
	if err := validateDerivativesCancelOnDisconnectScope(scope); err != nil {
		return err
	}
	return w.e.GlobalDerivatives().sendOKRequest(ctx, true, "private/disable_cancel_on_disconnect", &DerivativesCancelOnDisconnectWireRequest{Scope: scope})
}

// EnableCancelOnDisconnect has the gateway cancel the orders a connection placed once it closes, goes ten minutes
// without traffic or misses a heartbeat, but not once it closes with Logout; the connection scope, which an empty scope
// defaults to, applies to this connection, and the account scope to every connection opened afterwards. Orders placed
// by other connections are unaffected. The gateway documents it as websocket only
func (w WebsocketAPI) EnableCancelOnDisconnect(ctx context.Context, scope string) error {
	if err := validateDerivativesCancelOnDisconnectScope(scope); err != nil {
		return err
	}
	return w.e.GlobalDerivatives().sendOKRequest(ctx, true, "private/enable_cancel_on_disconnect", &DerivativesCancelOnDisconnectWireRequest{Scope: scope})
}

// GetCancelOnDisconnect returns whether this connection, which an empty scope defaults to, or the account cancels a
// connection's orders once it closes
func (w WebsocketAPI) GetCancelOnDisconnect(ctx context.Context, scope string) (*DerivativesCancelOnDisconnectResponse, error) {
	return w.e.GlobalDerivatives().getCancelOnDisconnect(ctx, true, scope)
}

// DisableHeartbeat stops the gateway sending heartbeats, so a quiet connection's traffic stops and the websocket
// manager may reconnect it once its traffic timeout passes; the gateway only accepts it over the websocket
func (w WebsocketAPI) DisableHeartbeat(ctx context.Context) error {
	return w.e.GlobalDerivatives().sendOKRequest(ctx, true, "public/disable_heartbeat", nil)
}

// SetHeartbeat has the gateway send heartbeats and test requests at an interval of at least 10 whole seconds, which
// the gateway refuses otherwise, closing the connection when a test request goes unanswered. The connection asks for
// a 15 second interval itself whenever it connects, replacing an interval set here; the gateway only accepts it over
// the websocket
func (w WebsocketAPI) SetHeartbeat(ctx context.Context, interval time.Duration) error {
	if interval < derivativesMinimumHeartbeatInterval {
		return fmt.Errorf("%w: %s", errDerivativesHeartbeatIntervalTooShort, interval)
	}
	if interval%time.Second != 0 {
		return fmt.Errorf("%w: %s", errDerivativesHeartbeatIntervalFraction, interval)
	}
	return w.e.GlobalDerivatives().sendOKRequest(ctx, true, "public/set_heartbeat", &DerivativesHeartbeatWireRequest{Interval: interval.Seconds()})
}
