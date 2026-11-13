package kraken

import (
	"context"
	"net/http"
	"net/url"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFuturesFills calls Get your fills, returning the account's last 100 fills on every contract, or with LastFillTime
// set the 100 before it. A nil request returns the last 100
func (e *Exchange) GetFuturesFills(ctx context.Context, req *FuturesFillsRequest) (*FuturesFillsResponse, error) {
	params := url.Values{}
	if req != nil && !req.LastFillTime.IsZero() {
		// Paging starts from a fill time Kraken sent, so the time is sent back in the millisecond precision it came in
		params.Set("lastFillTime", req.LastFillTime.UTC().Format("2006-01-02T15:04:05.000Z07:00"))
	}
	var resp *FuturesFillsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/fills", params, nil, &resp)
}
