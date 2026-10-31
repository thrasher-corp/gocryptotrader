package kraken

import (
	"context"
	"net/http"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// CheckFuturesAPIKey calls Check v3 API key, returning the details and permissions of the key signing the request
func (e *Exchange) CheckFuturesAPIKey(ctx context.Context) (*FuturesCheckAPIKeyResponse, error) {
	var resp *FuturesCheckAPIKeyResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/auth/v1/api-keys/v3/check", nil, nil, &resp)
}
