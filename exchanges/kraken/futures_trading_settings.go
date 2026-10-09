package kraken

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFuturesSelfTradeStrategy calls Get self trade strategy
func (e *Exchange) GetFuturesSelfTradeStrategy(ctx context.Context) (*FuturesSelfTradeStrategyResponse, error) {
	var resp *FuturesSelfTradeStrategyResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/self-trade-strategy", nil, nil, &resp)
}

// UpdateFuturesSelfTradeStrategy calls Update self trade strategy, setting the account-wide strategy to one of the values
// FuturesSelfTradeStrategyResponse's Strategy describes
func (e *Exchange) UpdateFuturesSelfTradeStrategy(ctx context.Context, strategy string) (*FuturesSelfTradeStrategyResponse, error) {
	if strategy == "" {
		return nil, errFuturesSelfTradeStrategyEmpty
	}
	params := url.Values{}
	params.Set("strategy", strategy)
	var resp *FuturesSelfTradeStrategyResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPut, "/api/v3/self-trade-strategy", params, nil, &resp)
}

// GetFuturesOffBookMaxLeverageCap calls Get the off-book max leverage cap. Only master accounts may call it
func (e *Exchange) GetFuturesOffBookMaxLeverageCap(ctx context.Context) (*FuturesOffBookMaxLeverageResponse, error) {
	var resp *FuturesOffBookMaxLeverageResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/rfq-assignment/max-leverage", nil, nil, &resp)
}

// SetFuturesOffBookMaxLeverageCap calls Set the off-book max leverage cap, returning the stored cap. maxLeverage is 0
// to 100 with at most 2 decimal places, and 0 opts the account out of every exposure-increasing assignment and RFQ
// fill without withdrawing its assignment preferences. Only master accounts may set it
func (e *Exchange) SetFuturesOffBookMaxLeverageCap(ctx context.Context, maxLeverage float64) (*FuturesOffBookMaxLeverageResponse, error) {
	if maxLeverage < 0 || maxLeverage > 100 {
		return nil, fmt.Errorf("%w: %v is outside 0 to 100", errFuturesInvalidMaxLeverage, maxLeverage)
	}
	value := strconv.FormatFloat(maxLeverage, 'f', -1, 64)
	if _, decimals, ok := strings.Cut(value, "."); ok && len(decimals) > 2 {
		return nil, fmt.Errorf("%w: %s has more than 2 decimal places", errFuturesInvalidMaxLeverage, value)
	}
	params := url.Values{}
	params.Set("maxLeverage", value)
	var resp *FuturesOffBookMaxLeverageResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPut, "/api/v3/rfq-assignment/max-leverage", params, nil, &resp)
}

// ClearFuturesOffBookMaxLeverageCap calls Clear the off-book max leverage cap, lifting the cap entirely, unlike a cap
// of 0, which blocks every exposure-increasing off-book fill. Only master accounts may clear it
func (e *Exchange) ClearFuturesOffBookMaxLeverageCap(ctx context.Context) error {
	return e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodDelete, "/api/v3/rfq-assignment/max-leverage", nil, nil, nil)
}
