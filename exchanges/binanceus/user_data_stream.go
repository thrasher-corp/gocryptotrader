package binanceus

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var errListenKeyRequired = errors.New("listen key required")

// CreateUserDataStream starts a user data stream and returns its listen key, which expires after 60 minutes unless
// ExtendUserDataStream keeps it alive. An account with an active listen key gets that key back, valid for another 60
// minutes
//
// Deprecated: Binance.US is retiring listen keys, targeting Q4 2026 for a date it has yet to confirm. Subscribe to the
// User Data Stream over the WebSocket API instead, with its userDataStream.subscribe.signature request
func (e *Exchange) CreateUserDataStream(ctx context.Context) (*ListenKeyResponse, error) {
	return e.createUserDataStream(ctx, false)
}

func (e *Exchange) createUserDataStream(ctx context.Context, useWebsocket bool) (*ListenKeyResponse, error) {
	var resp *ListenKeyResponse
	return resp, e.sendListenKeyRequest(ctx, useWebsocket, http.MethodPost, "userDataStream.start", nil, &resp)
}

// ExtendUserDataStream keeps a user data stream alive, extending its listen key's validity to 60 minutes from now.
// Binance.US recommends extending a listen key about every 30 minutes
//
// Deprecated: Binance.US is retiring listen keys, targeting Q4 2026 for a date it has yet to confirm. Subscribe to the
// User Data Stream over the WebSocket API instead, with its userDataStream.subscribe.signature request
func (e *Exchange) ExtendUserDataStream(ctx context.Context, listenKey string) error {
	return e.extendUserDataStream(ctx, listenKey, false)
}

func (e *Exchange) extendUserDataStream(ctx context.Context, listenKey string, useWebsocket bool) error {
	if listenKey == "" {
		return errListenKeyRequired
	}
	params := url.Values{}
	params.Set("listenKey", listenKey)
	return e.sendListenKeyRequest(ctx, useWebsocket, http.MethodPut, "userDataStream.ping", params, nil)
}

// CloseUserDataStream closes a user data stream and invalidates its listen key
//
// Deprecated: Binance.US is retiring listen keys, targeting Q4 2026 for a date it has yet to confirm. Subscribe to the
// User Data Stream over the WebSocket API instead, with its userDataStream.subscribe.signature request
func (e *Exchange) CloseUserDataStream(ctx context.Context, listenKey string) error {
	return e.closeUserDataStream(ctx, listenKey, false)
}

func (e *Exchange) closeUserDataStream(ctx context.Context, listenKey string, useWebsocket bool) error {
	if listenKey == "" {
		return errListenKeyRequired
	}
	params := url.Values{}
	params.Set("listenKey", listenKey)
	return e.sendListenKeyRequest(ctx, useWebsocket, http.MethodDelete, "userDataStream.stop", params, nil)
}

// sendListenKeyRequest sends a listen key request, which carries the API key without a signature, over REST as an
// httpMethod request, or with useWebsocket over the WebSocket API as method
func (e *Exchange) sendListenKeyRequest(ctx context.Context, useWebsocket bool, httpMethod, method string, params url.Values, result any) error {
	if useWebsocket {
		return e.SendAPIKeyWebsocketAPIRequest(ctx, defaultRate, method, params, result)
	}
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, httpMethod, common.EncodeURLValues("/api/v3/userDataStream", params), defaultRate, result)
}
