package bingx

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var errListenKeyEmpty = errors.New("listen key empty")

// GenerateListenKey creates a listen key for the account data streams, valid for 60 minutes unless extended
func (e *Exchange) GenerateListenKey(ctx context.Context) (string, error) {
	return e.generateListenKey(ctx, exchange.RestSpot)
}

// generateListenKey creates a listen key through the REST endpoint that serves a product's account streams, as a
// demo host's streams take keys from that host
func (e *Exchange) generateListenKey(ctx context.Context, ePath exchange.URL) (string, error) {
	var resp *ListenKeyResponse
	if err := e.SendAPIKeyHTTPRequest(request.WithRetryNotAllowed(ctx), ePath, http.MethodPost, "/openApi/user/auth/userDataStream", nil, generateListenKeyRate, &resp); err != nil {
		return "", err
	}
	if resp == nil || resp.ListenKey == "" {
		return "", errListenKeyEmpty
	}
	return resp.ListenKey, nil
}

// ExtendListenKey extends a listen key's validity by 60 minutes; BingX recommends extending every 30 minutes
func (e *Exchange) ExtendListenKey(ctx context.Context, listenKey string) error {
	return e.extendListenKey(ctx, exchange.RestSpot, listenKey)
}

func (e *Exchange) extendListenKey(ctx context.Context, ePath exchange.URL, listenKey string) error {
	if listenKey == "" {
		return errListenKeyEmpty
	}
	params := url.Values{}
	params.Set("listenKey", listenKey)
	// BingX documents an empty 204 reply as "no request parameters", so a reply must carry a body to be a success
	var resp json.RawMessage
	return e.SendAPIKeyHTTPRequest(ctx, ePath, http.MethodPut, "/openApi/user/auth/userDataStream", params, extendListenKeyRate, &resp)
}

// CloseListenKey closes a listen key, ending the account data streams that use it
func (e *Exchange) CloseListenKey(ctx context.Context, listenKey string) error {
	if listenKey == "" {
		return errListenKeyEmpty
	}
	params := url.Values{}
	params.Set("listenKey", listenKey)
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/openApi/user/auth/userDataStream", params, closeListenKeyRate, nil)
}
