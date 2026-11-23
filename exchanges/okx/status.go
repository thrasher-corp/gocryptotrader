package okx

import (
	"context"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GetSystemStatus retrieves system maintenance events in a state; without one OKX returns the scheduled, ongoing and
// pre-open events
func (e *Exchange) GetSystemStatus(ctx context.Context, state string) ([]SystemStatusResponse, error) {
	params := url.Values{}
	if state != "" {
		params.Set("state", state)
	}
	var resp []SystemStatusResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getEventStatusEPL, http.MethodGet, common.EncodeURLValues("system/status", params), nil, &resp, request.UnauthenticatedRequest)
}
