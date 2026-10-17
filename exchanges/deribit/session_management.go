package deribit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// Cancel on disconnect scopes; the connection scope can only be used over the websocket
const (
	CancelOnDisconnectScopeConnection = "connection"
	CancelOnDisconnectScopeAccount    = "account"
)

// minimumHeartbeatInterval is the shortest heartbeat interval Deribit accepts
const minimumHeartbeatInterval = 10 * time.Second

var (
	errHeartbeatIntervalTooShort      = errors.New("heartbeat interval must be at least 10 seconds")
	errInvalidCancelOnDisconnectScope = errors.New("invalid cancel on disconnect scope")
)

// GetCancelOnDisconnect returns whether the account cancels a connection's orders once it closes; over HTTP only the
// account scope can be read
func (e *Exchange) GetCancelOnDisconnect(ctx context.Context) (*CancelOnDisconnectResponse, error) {
	return e.getCancelOnDisconnect(ctx, CancelOnDisconnectScopeAccount, false)
}

func (e *Exchange) getCancelOnDisconnect(ctx context.Context, scope string, overWebsocket bool) (*CancelOnDisconnectResponse, error) {
	if err := validateCancelOnDisconnectScope(scope); err != nil {
		return nil, err
	}
	var resp *CancelOnDisconnectResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_cancel_on_disconnect", &cancelOnDisconnectParams{Scope: scope}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validateCancelOnDisconnectScope checks a cancel on disconnect scope is empty, which Deribit defaults to the
// connection scope, or a valid scope
func validateCancelOnDisconnectScope(scope string) error {
	switch scope {
	case "", CancelOnDisconnectScopeConnection, CancelOnDisconnectScopeAccount:
		return nil
	default:
		return fmt.Errorf("%w: %q", errInvalidCancelOnDisconnectScope, scope)
	}
}
