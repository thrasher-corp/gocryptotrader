package bitstamp

import (
	"context"
	"net/http"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// RevokeAllAPIKeys revokes every API key across all of the user's accounts, including the key making the request
func (e *Exchange) RevokeAllAPIKeys(ctx context.Context) (*RevokeAPIKeysResponse, error) {
	var resp *RevokeAPIKeysResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/revoke_all_api_keys/", nil, nil, &resp)
}
