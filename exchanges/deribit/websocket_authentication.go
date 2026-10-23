package deribit

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Authenticate requests an OAuth access token as Exchange.Authenticate does, over the websocket, which authenticates
// the connection for private requests
func (w WebsocketAPI) Authenticate(ctx context.Context, arg *AuthenticateRequest) (*AccessTokenResponse, error) {
	resp, err := w.e.authenticate(ctx, arg, true)
	if err != nil {
		return nil, err
	}
	// The connection is authenticated now, whether or not it was when it connected, so its private methods are sent
	w.e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return resp, nil
}

// ExchangeToken returns a token for another subject as Exchange.ExchangeToken does, over the websocket
func (w WebsocketAPI) ExchangeToken(ctx context.Context, arg *ExchangeTokenRequest) (*AccessTokenResponse, error) {
	return w.e.exchangeToken(ctx, arg, true)
}

// ForkToken returns a token for a new named session as Exchange.ForkToken does, over the websocket
func (w WebsocketAPI) ForkToken(ctx context.Context, arg *ForkTokenRequest) (*AccessTokenResponse, error) {
	return w.e.forkToken(ctx, arg, true)
}

// Logout closes the websocket connection without cancelling its orders, even with cancel on disconnect enabled, and
// invalidates the session's tokens when invalidateToken is set; Deribit closes the connection without a response, and
// only accepts it over the websocket
func (w WebsocketAPI) Logout(ctx context.Context, invalidateToken bool) error {
	if !w.e.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%w: private/logout", errWebsocketNotAuthenticated)
	}
	if !w.e.websocketDialled() {
		return fmt.Errorf("%w: private/logout", websocket.ErrNotConnected)
	}
	if err := w.e.Requester.InitiateRateLimit(ctx, nonMatchingEPL); err != nil {
		return err
	}
	return w.e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, &JSONRPCRequest{
		JSONRPC: jsonRPCVersion,
		ID:      w.e.MessageSequence(),
		Method:  "private/logout",
		Params:  &LogoutRequest{InvalidateToken: invalidateToken},
	})
}
