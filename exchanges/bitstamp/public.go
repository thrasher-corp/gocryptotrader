package bitstamp

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// ohlcSteps are the candle intervals supported by the OHLC endpoint
var ohlcSteps = []kline.Interval{kline.OneMin, kline.ThreeMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour, kline.FourHour, kline.SixHour, kline.TwelveHour, kline.OneDay, kline.ThreeDay}

// GetCurrencies returns all listed currencies with their deposit and withdrawal networks
func (e *Exchange) GetCurrencies(ctx context.Context) ([]CurrencyResponse, error) {
	var resp []CurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/currencies/", nil, &resp)
}

// GetTickers returns ticker data for all markets
func (e *Exchange) GetTickers(ctx context.Context) ([]TickerResponse, error) {
	var resp []TickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/ticker/", nil, &resp)
}

// GetTicker returns ticker data for a market
func (e *Exchange) GetTicker(ctx context.Context, pair currency.Pair) (*TickerResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *TickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/ticker/"+formatMarketSymbol(pair)+"/", nil, &resp)
}

// GetHourlyTicker returns ticker data for a market over the last hour
// The open, high, low, volume and volume weighted average price cover the last hour, and the 24 hour fields are unset
func (e *Exchange) GetHourlyTicker(ctx context.Context, pair currency.Pair) (*TickerResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *TickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/ticker_hour/"+formatMarketSymbol(pair)+"/", nil, &resp)
}

// GetOrderbook returns the orderbook for a market
func (e *Exchange) GetOrderbook(ctx context.Context, pair currency.Pair, grouping OrderbookGrouping) (*OrderbookResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if grouping > OrderbookUngroupedWithOrderIDs {
		return nil, fmt.Errorf("%w: %d", errInvalidOrderbookGrouping, grouping)
	}
	params := url.Values{}
	params.Set("group", strconv.FormatUint(uint64(grouping), 10))
	var resp *OrderbookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/order_book/"+formatMarketSymbol(pair)+"/", params, &resp)
}

// GetTransactions returns public trades for a market
// timePeriod may be TransactionPeriodMinute, TransactionPeriodHour or TransactionPeriodDay, and defaults to an hour when empty
func (e *Exchange) GetTransactions(ctx context.Context, pair currency.Pair, timePeriod string) ([]TransactionResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	switch timePeriod {
	case "":
	case TransactionPeriodMinute, TransactionPeriodHour, TransactionPeriodDay:
		params.Set("time", timePeriod)
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidTransactionPeriod, timePeriod)
	}
	var resp []TransactionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/transactions/"+formatMarketSymbol(pair)+"/", params, &resp)
}

// GetMarkets returns all available markets
// entity optionally restricts the markets to those available to a broker entity, e.g. EntityEuropeSA
func (e *Exchange) GetMarkets(ctx context.Context, entity string) ([]MarketResponse, error) {
	params := url.Values{}
	if entity != "" {
		params.Set("entity", entity)
	}
	var resp []MarketResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/markets/", params, &resp)
}

// GetOHLC returns candle data for a market
func (e *Exchange) GetOHLC(ctx context.Context, req *OHLCRequest) (*OHLCResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(ohlcSteps, req.Step) {
		return nil, fmt.Errorf("%w: %s", errInvalidInterval, req.Step)
	}
	if req.Limit == 0 || req.Limit > maxResultLimit {
		return nil, fmt.Errorf("%w: %d must be between 1 and %d", errInvalidLimit, req.Limit, maxResultLimit)
	}
	if !req.Start.IsZero() && !req.End.IsZero() && req.Start.After(req.End) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("step", strconv.FormatInt(int64(req.Step.Duration()/time.Second), 10))
	params.Set("limit", strconv.FormatUint(req.Limit, 10))
	if !req.Start.IsZero() {
		params.Set("start", strconv.FormatInt(req.Start.Unix(), 10))
	}
	if !req.End.IsZero() {
		params.Set("end", strconv.FormatInt(req.End.Unix(), 10))
	}
	if req.ExcludeCurrentCandle {
		params.Set("exclude_current_candle", "true")
	}
	var resp *OHLCResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/ohlc/"+formatMarketSymbol(req.Pair)+"/", params, &resp)
}

// GetEURUSDConversionRate returns the EUR/USD conversion rate
func (e *Exchange) GetEURUSDConversionRate(ctx context.Context) (*EURUSDConversionRateResponse, error) {
	var resp *EURUSDConversionRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/eur_usd/", nil, &resp)
}

// GetFundingRate returns the current funding rate for a perpetual market
func (e *Exchange) GetFundingRate(ctx context.Context, pair currency.Pair) (*FundingRateResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *FundingRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/funding_rate/"+formatMarketSymbol(pair)+"/", nil, &resp)
}

// GetFundingRateHistory returns up to 100 funding rates for a perpetual market, oldest first
// Limit keeps the most recent rates within the time range, so earlier rates are fetched by moving Until back. The API
// documents rates from up to 30 days ago, but returns older rates for a time range which reaches them
func (e *Exchange) GetFundingRateHistory(ctx context.Context, req *FundingRateHistoryRequest) (*FundingRateHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Limit > maxFundingRateHistoryLimit {
		return nil, fmt.Errorf("%w: %d must not exceed %d", errInvalidLimit, req.Limit, maxFundingRateHistoryLimit)
	}
	if !req.Since.IsZero() && !req.Until.IsZero() && req.Since.After(req.Until) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if !req.Since.IsZero() {
		params.Set("since_timestamp", strconv.FormatInt(req.Since.Unix(), 10))
	}
	if !req.Until.IsZero() {
		params.Set("until_timestamp", strconv.FormatInt(req.Until.Unix(), 10))
	}
	var resp *FundingRateHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/funding_rate_history/"+formatMarketSymbol(req.Pair)+"/", params, &resp)
}

// GetVirtualAssetServiceProviders returns the Virtual Asset Service Providers used for Travel Rule compliance
func (e *Exchange) GetVirtualAssetServiceProviders(ctx context.Context, req *PaginationRequest) (*VirtualAssetServiceProvidersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := paginationParams(req)
	var resp *VirtualAssetServiceProvidersResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/travel_rule/vasps/", params, &resp)
}
