package coinbase

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// Gateway cancel on disconnect scopes: the connection scope applies to one websocket connection, and is only usable
// over it, while the account scope applies to every connection opened afterwards
const (
	DerivativesCancelOnDisconnectScopeConnection = "connection"
	DerivativesCancelOnDisconnectScopeAccount    = "account"
)

var errDerivativesInvalidCancelOnDisconnectScope = errors.New("invalid gateway cancel on disconnect scope")

// GetCancelOnDisconnect returns whether the account cancels a connection's orders once it closes; over HTTP only the
// account scope can be read, which WebsocketAPI.GetCancelOnDisconnect reads with the connection's
func (g GlobalDerivativesAPI) GetCancelOnDisconnect(ctx context.Context) (*DerivativesCancelOnDisconnectResponse, error) {
	return g.getCancelOnDisconnect(ctx, false, DerivativesCancelOnDisconnectScopeAccount)
}

func (g GlobalDerivativesAPI) getCancelOnDisconnect(ctx context.Context, overWebsocket bool, scope string) (*DerivativesCancelOnDisconnectResponse, error) {
	if err := validateDerivativesCancelOnDisconnectScope(scope); err != nil {
		return nil, err
	}
	var resp *DerivativesCancelOnDisconnectResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_cancel_on_disconnect", &DerivativesCancelOnDisconnectWireRequest{Scope: scope}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validateDerivativesCancelOnDisconnectScope checks a cancel on disconnect scope is empty, which the gateway defaults
// to the connection scope, or one the gateway documents
func validateDerivativesCancelOnDisconnectScope(scope string) error {
	switch scope {
	case "", DerivativesCancelOnDisconnectScopeConnection, DerivativesCancelOnDisconnectScopeAccount:
		return nil
	default:
		return fmt.Errorf("%w: %q", errDerivativesInvalidCancelOnDisconnectScope, scope)
	}
}
