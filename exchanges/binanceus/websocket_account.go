package binanceus

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// GetAccount returns the account's information and balances as Exchange.GetAccount does, over the WebSocket API
func (w WebsocketAPI) GetAccount(ctx context.Context) (*AccountResponse, error) {
	return w.e.getAccount(ctx, true)
}

// GetMyFilters returns the filters that apply to the account on a symbol as Exchange.GetMyFilters does, over the
// WebSocket API
func (w WebsocketAPI) GetMyFilters(ctx context.Context, symbol currency.Pair) (*MyFiltersResponse, error) {
	return w.e.getMyFilters(ctx, symbol, true)
}
