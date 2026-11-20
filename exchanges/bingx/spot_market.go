package bingx

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	errSpotOrderBookAggregationEmpty = errors.New("order book aggregation type empty")
	errSpotKlineRowLength            = errors.New("kline row too short")
)

// GetSpotSymbols returns the spot trading symbols and their trading rules, or only symbol's when it is set. An unknown
// symbol returns no symbols
func (e *Exchange) GetSpotSymbols(ctx context.Context, symbol currency.Pair) ([]SpotSymbolResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		if err := e.spotSetSymbol(params, symbol); err != nil {
			return nil, err
		}
	}
	var resp *SpotSymbolsResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/openApi/spot/v1/common/symbols", params), spotSymbolsRate, &resp); err != nil {
		return nil, err
	}
	return resp.Symbols, nil
}

// GetSpotRecentTrades returns a spot symbol's most recent trades, newest first. limit defaults to 100 and is at most
// 500
func (e *Exchange) GetSpotRecentTrades(ctx context.Context, symbol currency.Pair, limit uint64) ([]SpotRecentTradeResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, symbol); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []SpotRecentTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/openApi/spot/v2/quote/trades", params), spotRecentTradesRate, &resp)
}

// GetSpotOrderBook returns a spot symbol's order book. limit defaults to 20 levels a side and is at most 1000. Asks
// arrive in descending price order, the best ask last
func (e *Exchange) GetSpotOrderBook(ctx context.Context, symbol currency.Pair, limit uint64) (*SpotOrderBookResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, symbol); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *SpotOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/openApi/spot/v1/market/depth", params), spotOrderBookRate, &resp)
}

// GetSpotKlines returns a spot symbol's candles, newest first. Limit defaults to 500 and is at most 1440. Candle periods
// start from UTC unless TimeZone asks for UTC+8
func (e *Exchange) GetSpotKlines(ctx context.Context, req *SpotKlinesRequest) ([]SpotKlineResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Interval == 0 {
		return nil, kline.ErrInvalidInterval
	}
	interval, err := intervalToString(req.Interval)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, req.Symbol); err != nil {
		return nil, err
	}
	params.Set("interval", interval)
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params.Set("timeZone", strconv.FormatUint(uint64(req.TimeZone), 10))
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []SpotKlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/openApi/spot/v2/quote/klines", params), spotKlinesRate, &resp)
}

// GetSpotTicker returns the price change statistics over the last 24 hours of a spot symbol, or of every symbol when
// symbol is empty
func (e *Exchange) GetSpotTicker(ctx context.Context, symbol currency.Pair) ([]SpotTickerResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		if err := e.spotSetSymbol(params, symbol); err != nil {
			return nil, err
		}
	}
	var resp []SpotTickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/openApi/spot/v2/quote/ticker", params), spotTickerRate, &resp)
}

// GetSpotAggregatedOrderBook returns a spot symbol's order book with its levels merged to the price precision Type
// selects. Limit defaults to 20 levels a side
func (e *Exchange) GetSpotAggregatedOrderBook(ctx context.Context, req *SpotAggregatedOrderBookRequest) (*SpotAggregatedOrderBookResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Type == "" {
		return nil, errSpotOrderBookAggregationEmpty
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, req.Symbol); err != nil {
		return nil, err
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	params.Set("type", string(req.Type))
	var resp *SpotAggregatedOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/openApi/spot/v2/quote/depth", params), spotAggregatedOrderBookRate, &resp)
}

// GetSpotPriceTicker returns the latest price of a spot symbol, or of every symbol when symbol is empty
func (e *Exchange) GetSpotPriceTicker(ctx context.Context, symbol currency.Pair) ([]SpotPriceTickerResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		if err := e.spotSetSymbol(params, symbol); err != nil {
			return nil, err
		}
	}
	var resp []SpotPriceTickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/openApi/spot/v2/quote/price", params), spotPriceTickerRate, &resp)
}

// GetSpotBookTicker returns the best bid and ask prices and quantities of a spot symbol, or of every symbol when symbol
// is empty
func (e *Exchange) GetSpotBookTicker(ctx context.Context, symbol currency.Pair) ([]SpotBookTickerResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		if err := e.spotSetSymbol(params, symbol); err != nil {
			return nil, err
		}
	}
	var resp []SpotBookTickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/openApi/spot/v2/quote/bookTicker", params), spotBookTickerRate, &resp)
}

// GetSpotHistoricalKlines returns a spot symbol's older candles: newest first without a time range and oldest first
// with one. Limit defaults to 500 and is at most 1000. Daily candles start from UTC+8 midnight, and a recent range
// returns code 100204, no data found: GetSpotKlines serves recent candles
func (e *Exchange) GetSpotHistoricalKlines(ctx context.Context, req *SpotHistoricalKlinesRequest) ([]SpotKlineResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Interval == 0 {
		return nil, kline.ErrInvalidInterval
	}
	interval, err := intervalToString(req.Interval)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, req.Symbol); err != nil {
		return nil, err
	}
	params.Set("interval", interval)
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []SpotKlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/openApi/spot/v2/quote/historicalKlines", params), spotHistoricalKlinesRate, &resp)
}

// GetSpotHistoricalTrades returns a spot symbol's trades, newest first: the most recent, or those older than FromID.
// Limit defaults to 100 and is at most 1000
func (e *Exchange) GetSpotHistoricalTrades(ctx context.Context, req *SpotHistoricalTradesRequest) ([]SpotHistoricalTradeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.spotSetSymbol(params, req.Symbol); err != nil {
		return nil, err
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.FromID != "" {
		params.Set("fromId", req.FromID)
	}
	var resp []SpotHistoricalTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/openApi/spot/v2/quote/historicalTrades", params), spotHistoricalTradesRate, &resp)
}

// spotSetSymbol sets params' symbol to pair in the spot request format
func (e *Exchange) spotSetSymbol(params url.Values, pair currency.Pair) error {
	symbol, err := e.FormatSymbol(pair, asset.Spot)
	if err != nil {
		return err
	}
	params.Set("symbol", symbol)
	return nil
}
