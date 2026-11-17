package bingx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// GetSwapContracts returns the USDT- and USDC-settled perpetual contracts from USDT-M Perp Futures symbols, or the one
// a symbol names
func (e *Exchange) GetSwapContracts(ctx context.Context, symbol currency.Pair) ([]SwapContractResponse, error) {
	params, err := e.swapMarketSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp []SwapContractResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v2/quote/contracts", params), swapContractsRate, &resp)
}

// GetSwapOrderBook returns a perpetual contract's order book from Order Book. Valid limits are 5, 10, 20, 50, 100, 500
// and 1000; 0 means the default of 20
func (e *Exchange) GetSwapOrderBook(ctx context.Context, symbol currency.Pair, limit uint64) (*SwapOrderBookResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.swapMarketSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *SwapOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v2/quote/depth", params), swapOrderBookRate, &resp)
}

// GetSwapRecentTrades returns a perpetual contract's latest market trades from Recent Trades List, oldest first. The
// limit is documented as 500 when zero and at most 1000, but BingX returns 1000 trades whatever the limit
func (e *Exchange) GetSwapRecentTrades(ctx context.Context, symbol currency.Pair, limit uint64) ([]SwapRecentTradeResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.swapMarketSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []SwapRecentTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v2/quote/trades", params), swapRecentTradesRate, &resp)
}

// GetSwapMarkPriceAndFundingRate returns the mark price and funding rate of one perpetual contract or, with an empty
// symbol, every contract, from Mark Price and Funding Rate
func (e *Exchange) GetSwapMarkPriceAndFundingRate(ctx context.Context, symbol currency.Pair) ([]SwapMarkPriceAndFundingRateResponse, error) {
	params, err := e.swapMarketSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp objectOrArray[SwapMarkPriceAndFundingRateResponse]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v2/quote/premiumIndex", params), swapMarkPriceAndFundingRateRate, &resp)
}

// GetSwapFundingRateHistory returns a perpetual contract's completed funding settlements from Get Funding Rate History,
// newest first: the latest of a time range, which includes both of its bounds, or without one the latest of all
func (e *Exchange) GetSwapFundingRateHistory(ctx context.Context, arg *SwapFundingRateHistoryRequest) ([]SwapFundingRateHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.swapMarketSymbolParams(arg.Symbol)
	if err != nil {
		return nil, err
	}
	if err := setTimeRange(params, "startTime", "endTime", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []SwapFundingRateHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v2/quote/fundingRate", params), swapFundingRateHistoryRate, &resp)
}

// GetSwapKlines returns a perpetual contract's candles from Kline/Candlestick Data, newest first. A time range includes
// its start and excludes its end
func (e *Exchange) GetSwapKlines(ctx context.Context, arg *SwapKlinesRequest) ([]SwapKlineResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.swapMarketKlineParams(arg.Symbol, arg.Interval, arg.StartTime, arg.EndTime, arg.Limit)
	if err != nil {
		return nil, err
	}
	if arg.TimeZone > 0 {
		params.Set("timeZone", strconv.FormatUint(arg.TimeZone, 10))
	}
	var resp []SwapKlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v3/quote/klines", params), swapKlinesRate, &resp)
}

// GetSwapOpenInterest returns a perpetual contract's open interest from Open Interest Statistics
func (e *Exchange) GetSwapOpenInterest(ctx context.Context, symbol currency.Pair) (*SwapOpenInterestResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.swapMarketSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp *SwapOpenInterestResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v2/quote/openInterest", params), swapOpenInterestRate, &resp)
}

// GetSwap24HourTicker returns the price change statistics over 24 hours of one perpetual contract or, with an empty
// symbol, every contract, from 24hr Ticker Price Change Statistics
func (e *Exchange) GetSwap24HourTicker(ctx context.Context, symbol currency.Pair) ([]Swap24HourTickerResponse, error) {
	params, err := e.swapMarketSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp objectOrArray[Swap24HourTickerResponse]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v2/quote/ticker", params), swapTickerRate, &resp)
}

// GetSwapHistoricalTrades returns a perpetual contract's market trades from Query historical transaction orders. The
// trade ID to start from and the limit are documented, but BingX returns its latest 1000 trades whatever they are
func (e *Exchange) GetSwapHistoricalTrades(ctx context.Context, arg *SwapHistoricalTradesRequest) ([]SwapHistoricalTradeResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.swapMarketSymbolParams(arg.Symbol)
	if err != nil {
		return nil, err
	}
	if arg.FromID > 0 {
		params.Set("fromId", strconv.FormatUint(arg.FromID, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []SwapHistoricalTradeResponse
	return resp, e.SendTimestampedHTTPRequest(ctx, exchange.RestUSDTMargined, "/openApi/swap/v1/market/historicalTrades", params, swapHistoricalTradesRate, &resp)
}

// GetSwapOrderBookTicker returns a perpetual contract's best bid and ask from Symbol Order Book Ticker
func (e *Exchange) GetSwapOrderBookTicker(ctx context.Context, symbol currency.Pair) (*SwapOrderBookTickerResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.swapMarketSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp *SwapOrderBookTickerResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v2/quote/bookTicker", params), swapBookTickerRate, &resp); err != nil {
		return nil, err
	}
	if resp.BookTicker == nil {
		return nil, fmt.Errorf("%w: book_ticker", common.ErrNoResponse)
	}
	return resp, nil
}

// GetSwapMarkPriceKlines returns a perpetual contract's mark price candles from Mark Price Kline/Candlestick Data,
// newest first. A time range selects the candles that close within it
func (e *Exchange) GetSwapMarkPriceKlines(ctx context.Context, arg *SwapMarkPriceKlinesRequest) ([]SwapMarkPriceKlineResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.swapMarketKlineParams(arg.Symbol, arg.Interval, arg.StartTime, arg.EndTime, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []SwapMarkPriceKlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v1/market/markPriceKlines", params), swapMarkPriceKlinesRate, &resp)
}

// GetSwapPriceTicker returns the latest price of one perpetual contract or, with an empty symbol, every contract, from
// Symbol Price Ticker
func (e *Exchange) GetSwapPriceTicker(ctx context.Context, symbol currency.Pair) ([]SwapPriceTickerResponse, error) {
	params, err := e.swapMarketSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp objectOrArray[SwapPriceTickerResponse]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v1/ticker/price", params), swapPriceTickerRate, &resp)
}

// GetSwapTradingRules returns a perpetual contract's order size, price and open order limits from Trading Rules
func (e *Exchange) GetSwapTradingRules(ctx context.Context, symbol currency.Pair) (*SwapTradingRulesResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.swapMarketSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp *SwapTradingRulesResponse
	return resp, e.SendTimestampedHTTPRequest(ctx, exchange.RestUSDTMargined, "/openApi/swap/v1/tradingRules", params, swapTradingRulesRate, &resp)
}

// GetSwapTradingSchedule returns the trading schedule of one non-crypto (TradFi) perpetual contract or, with an empty
// symbol, every one, from Trading Schedule. A schedule covers the seven calendar days from today's start in UTC+8;
// BingX answers a crypto contract with code 109419
func (e *Exchange) GetSwapTradingSchedule(ctx context.Context, symbol currency.Pair) ([]SwapTradingScheduleResponse, error) {
	params, err := e.swapMarketSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp []SwapTradingScheduleResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/openApi/swap/v1/quote/tradingSchedule", params), swapTradingScheduleRate, &resp)
}

// swapMarketSymbolParams returns query parameters holding a pair as a perpetual symbol, or none for an empty pair, which
// the endpoints whose symbol is optional answer for every contract
func (e *Exchange) swapMarketSymbolParams(pair currency.Pair) (url.Values, error) {
	params := url.Values{}
	if pair.IsEmpty() {
		return params, nil
	}
	symbol, err := e.FormatSymbol(pair, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	return params, nil
}

// swapMarketKlineParams validates and returns the parameters the candle endpoints share
func (e *Exchange) swapMarketKlineParams(symbol currency.Pair, interval kline.Interval, startTime, endTime time.Time, limit uint64) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	intervalValue, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	params, err := e.swapMarketSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	if err := setTimeRange(params, "startTime", "endTime", startTime, endTime); err != nil {
		return nil, err
	}
	params.Set("interval", intervalValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params, nil
}
