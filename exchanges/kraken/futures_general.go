package kraken

import (
	"context"
	"net/http"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFuturesNotifications calls Get notifications, returning the platform's notifications
func (e *Exchange) GetFuturesNotifications(ctx context.Context) (*FuturesNotificationsResponse, error) {
	var resp *FuturesNotificationsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/notifications", nil, nil, &resp)
}
