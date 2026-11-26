package bingx

import (
	"context"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// GetCoinMarginedContracts returns the contract information of a coin-M perpetual, or of every coin-M perpetual when
// symbol is empty
func (e *Exchange) GetCoinMarginedContracts(ctx context.Context, symbol currency.Pair) ([]CoinMarginedContractResponse, error) {
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp []CoinMarginedContractResponse
	return resp, e.SendTimestampedHTTPRequest(ctx, exchange.RestCoinMargined, "/openApi/cswap/v1/market/contracts", params, coinMarginedContractsRate, &resp)
}

// GetCoinMarginedPriceAndFundingRate returns the mark price, index price and current funding rate of a coin-M
// perpetual, or of every coin-M perpetual when symbol is empty
func (e *Exchange) GetCoinMarginedPriceAndFundingRate(ctx context.Context, symbol currency.Pair) ([]CoinMarginedPriceAndFundingRateResponse, error) {
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp []CoinMarginedPriceAndFundingRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/openApi/cswap/v1/market/premiumIndex", params), coinMarginedPremiumIndexRate, &resp)
}

// GetCoinMarginedOpenInterest returns the open interest of a coin-M perpetual, or of every coin-M perpetual when symbol
// is empty, through Get Swap Open Positions
func (e *Exchange) GetCoinMarginedOpenInterest(ctx context.Context, symbol currency.Pair) ([]CoinMarginedOpenInterestResponse, error) {
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp []CoinMarginedOpenInterestResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/openApi/cswap/v1/market/openInterest", params), coinMarginedOpenInterestRate, &resp)
}

// GetCoinMarginedKlineData returns a coin-M perpetual's candles, newest first. BingX serves the last 30 days of candles
func (e *Exchange) GetCoinMarginedKlineData(ctx context.Context, req *CoinMarginedKlineDataRequest) ([]CoinMarginedKlineResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	interval, err := intervalToString(req.Interval)
	if err != nil {
		return nil, err
	}
	params, err := e.coinMarginedSymbolParams(req.Symbol)
	if err != nil {
		return nil, err
	}
	params.Set("interval", interval)
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.TimeZone != 0 {
		params.Set("timeZone", strconv.FormatUint(req.TimeZone, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CoinMarginedKlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/openApi/cswap/v1/market/klines", params), coinMarginedKlinesRate, &resp)
}

// GetCoinMarginedOrderBook returns a coin-M perpetual's order book through Query Depth Data. limit is 5, 10, 20, 50,
// 100, 500 or 1000 levels a side, and 20 when 0
func (e *Exchange) GetCoinMarginedOrderBook(ctx context.Context, symbol currency.Pair, limit uint64) (*CoinMarginedOrderBookResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *CoinMarginedOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/openApi/cswap/v1/market/depth", params), coinMarginedOrderBookRate, &resp)
}

// GetCoinMargined24HourPriceChange returns the 24 hour price change statistics of a coin-M perpetual, or of every
// coin-M perpetual when symbol is empty
func (e *Exchange) GetCoinMargined24HourPriceChange(ctx context.Context, symbol currency.Pair) ([]CoinMargined24HourPriceChangeResponse, error) {
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp []CoinMargined24HourPriceChangeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/openApi/cswap/v1/market/ticker", params), coinMarginedTickerRate, &resp)
}

// coinMarginedSymbolParams returns parameters holding a coin-M perpetual symbol, or no symbol when it is empty, which
// the endpoints taking an optional symbol read as every symbol
func (e *Exchange) coinMarginedSymbolParams(symbol currency.Pair) (url.Values, error) {
	params := url.Values{}
	if symbol.IsEmpty() {
		return params, nil
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", s)
	return params, nil
}
