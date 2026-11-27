package bitstamp

import (
	"context"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/exchange"
)

// RevokeAllAPIKeys revokes every API key across all of the user's accounts, including the key making the request
func (e *Exchange) RevokeAllAPIKeys(ctx context.Context) (*RevokeAPIKeysResponse, error) {
	var resp *RevokeAPIKeysResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/revoke_all_api_keys/", nil, nil, &resp)
}
