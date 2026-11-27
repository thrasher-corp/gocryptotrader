package deribit

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Authenticate requests an OAuth access token as Exchange.Authenticate does, over the websocket, which authenticates
// the request connection for private requests
func (w WebsocketAPI) Authenticate(ctx context.Context, arg *AuthenticateRequest) (*AccessTokenResponse, error) {
	conn, err := w.e.requestConnection("public/auth")
	if err != nil {
		return nil, err
	}
	return w.e.authenticateConnection(ctx, conn, arg)
}

// authenticateConnection requests an OAuth access token over a connection, which authenticates it for private requests
func (e *Exchange) authenticateConnection(ctx context.Context, conn websocket.Connection, arg *AuthenticateRequest) (*AccessTokenResponse, error) {
	params, err := arg.params()
	if err != nil {
		return nil, err
	}
	var resp *AccessTokenResponse
	if err := e.sendWebsocketRequest(ctx, conn, nonMatchingEPL, "public/auth", params, &resp); err != nil {
		return nil, err
	}
	token, err := accessTokenResult(resp)
	if err != nil {
		return nil, err
	}
	// The connection is authenticated now, whether or not authenticated requests were allowed when it connected, so its
	// private methods are sent
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return token, nil
}

// ExchangeToken returns a token for another subject as Exchange.ExchangeToken does, over the websocket
func (w WebsocketAPI) ExchangeToken(ctx context.Context, arg *ExchangeTokenRequest) (*AccessTokenResponse, error) {
	return w.e.exchangeToken(ctx, arg, true)
}

// ForkToken returns a token for a new named session as Exchange.ForkToken does, over the websocket
func (w WebsocketAPI) ForkToken(ctx context.Context, arg *ForkTokenRequest) (*AccessTokenResponse, error) {
	return w.e.forkToken(ctx, arg, true)
}

// Logout closes the request connection without cancelling its orders, even with cancel on disconnect enabled, and
// invalidates the session's tokens when invalidateToken is set; Deribit closes the connection without a response, after
// which the manager connects the websocket again. Deribit only accepts it over the websocket
func (w WebsocketAPI) Logout(ctx context.Context, invalidateToken bool) error {
	if !w.e.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%w: private/logout", errWebsocketNotAuthenticated)
	}
	conn, err := w.e.requestConnection("private/logout")
	if err != nil {
		return err
	}
	if err := w.e.Requester.InitiateRateLimit(ctx, nonMatchingEPL); err != nil {
		return err
	}
	return conn.SendJSONMessage(ctx, request.Unset, &JSONRPCRequest{
		JSONRPC: jsonRPCVersion,
		ID:      w.e.MessageSequence(),
		Method:  "private/logout",
		Params:  &LogoutRequest{InvalidateToken: invalidateToken},
	})
}
