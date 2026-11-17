package binanceus

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	errAggregatedTradesStartRequired = errors.New("fromID or start time required to collect more than 1000 aggregate trades")
	errInvalidWindowSize             = errors.New("invalid window size")
)

// tickerSymbolsLimit is the most symbols the rolling window and trading day tickers take in one request
const tickerSymbolsLimit = 100

var errTooManySymbols = errors.New("too many symbols")

// checkTickerSymbols refuses a rolling window or trading day ticker request for more symbols than one request takes
func checkTickerSymbols(symbols int) error {
	if symbols > tickerSymbolsLimit {
		return fmt.Errorf("%w: %d, at most %d", errTooManySymbols, symbols, tickerSymbolsLimit)
	}
	return nil
}

// GetMostRecentTrades returns the recent trades list of a symbol
func (e *Exchange) GetMostRecentTrades(ctx context.Context, req *RecentTradeRequest) ([]RecentTradeResponse, error) {
	return e.getMostRecentTrades(ctx, req, false)
}

func (e *Exchange) getMostRecentTrades(ctx context.Context, req *RecentTradeRequest, useWebsocket bool) ([]RecentTradeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []RecentTradeResponse
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/trades", "trades.recent", params, tradesRate, &resp)
}

// GetHistoricalTrades looks up older trades of a symbol, starting from fromID when it is set and otherwise returning the
// most recent trades; limit is 500 when zero and at most 1000
func (e *Exchange) GetHistoricalTrades(ctx context.Context, symbol currency.Pair, limit, fromID uint64) ([]RecentTradeResponse, error) {
	return e.getHistoricalTrades(ctx, symbol, limit, fromID, false)
}

func (e *Exchange) getHistoricalTrades(ctx context.Context, symbol currency.Pair, limit, fromID uint64, useWebsocket bool) ([]RecentTradeResponse, error) {
	symbolValue, err := e.formatSymbol(symbol)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if fromID > 0 {
		params.Set("fromId", strconv.FormatUint(fromID, 10))
	}
	var resp []RecentTradeResponse
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/historicalTrades", "trades.historical", params, tradesRate, &resp)
}

// GetAggregatedTrades returns the compressed/aggregate trades list. A symbol alone returns the most recent trades. A
// limit above the endpoint's maximum of 1000 is collected over several requests, paging on from FromID or StartTime
// until the limit or EndTime is reached
func (e *Exchange) GetAggregatedTrades(ctx context.Context, req *AggregatedTradeRequest) ([]AggregatedTradeResponse, error) {
	return e.getAggregatedTrades(ctx, req, false)
}

func (e *Exchange) getAggregatedTrades(ctx context.Context, req *AggregatedTradeRequest, useWebsocket bool) ([]AggregatedTradeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.FromID != 0 && (!req.StartTime.IsZero() || !req.EndTime.IsZero()) {
		return nil, fmt.Errorf("%w: fromId cannot be sent with startTime or endTime", errInvalidParameterCombination)
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.FromID != 0 {
		params.Set("fromId", strconv.FormatUint(req.FromID, 10))
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Limit > 1000 {
		if req.FromID == 0 && req.StartTime.IsZero() {
			return nil, errAggregatedTradesStartRequired
		}
		return e.batchAggregateTrades(ctx, req, params, useWebsocket)
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []AggregatedTradeResponse
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/aggTrades", "trades.aggregate", params, aggregatedTradesRate, &resp)
}

// batchAggregateTrades collects aggregate trades a thousand at a time. The first page starts at FromID or StartTime and
// each later page starts at the last trade collected, which the endpoint returns again since fromId is inclusive
func (e *Exchange) batchAggregateTrades(ctx context.Context, req *AggregatedTradeRequest, params url.Values, useWebsocket bool) ([]AggregatedTradeResponse, error) {
	params.Set("limit", "1000")
	var trades []AggregatedTradeResponse
	for uint64(len(trades)) < req.Limit {
		var page []AggregatedTradeResponse
		if err := e.sendPublicRequest(ctx, useWebsocket, "/api/v3/aggTrades", "trades.aggregate", params, aggregatedTradesRate, &page); err != nil {
			return nil, fmt.Errorf("error fetching aggregate trades for %s: %w", req.Symbol, err)
		}
		if len(trades) > 0 && len(page) > 0 && page[0].AggregateTradeID == trades[len(trades)-1].AggregateTradeID {
			page = page[1:]
		}
		if !req.EndTime.IsZero() {
			// Pages requested by fromId run past EndTime, which only the first request sends
			if afterEnd := sort.Search(len(page), func(i int) bool {
				return page[i].Timestamp.Time().After(req.EndTime)
			}); afterEnd < len(page) {
				trades = append(trades, page[:afterEnd]...)
				break
			}
		}
		if len(page) == 0 {
			break
		}
		trades = append(trades, page...)
		params.Del("startTime")
		params.Del("endTime")
		params.Set("fromId", strconv.FormatUint(trades[len(trades)-1].AggregateTradeID, 10))
	}
	if uint64(len(trades)) > req.Limit {
		trades = trades[:req.Limit]
	}
	return trades, nil
}

// GetOrderBook returns the order book of a symbol. A limit of zero returns the default 100 levels per side, and a limit
// above 5000 returns 5000
func (e *Exchange) GetOrderBook(ctx context.Context, req *OrderBookRequest) (*OrderBookResponse, error) {
	return e.getOrderBook(ctx, req, false)
}

func (e *Exchange) getOrderBook(ctx context.Context, req *OrderBookRequest, useWebsocket bool) (*OrderBookResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.SymbolStatus != "" {
		params.Set("symbolStatus", string(req.SymbolStatus))
	}
	var resp *OrderBookResponse
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/depth", "depth", params, orderbookLimit(req.Limit), &resp)
}

// GetSpotKline returns the kline/candlestick bars of a symbol
func (e *Exchange) GetSpotKline(ctx context.Context, req *KlinesRequest) ([]CandleStickResponse, error) {
	return e.getSpotKline(ctx, req, false)
}

func (e *Exchange) getSpotKline(ctx context.Context, req *KlinesRequest, useWebsocket bool) ([]CandleStickResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	interval, err := intervalToString(req.Interval)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.TimeZone != "" {
		params.Set("timeZone", req.TimeZone)
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CandleStickResponse
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/klines", "klines", params, klinesRate, &resp)
}

// GetLatestSpotPrice returns the symbol price ticker: the latest price of each symbol, or of every symbol when none are
// given
func (e *Exchange) GetLatestSpotPrice(ctx context.Context, symbols currency.Pairs, symbolStatus SymbolStatus) ([]SymbolPriceResponse, error) {
	return e.getLatestSpotPrice(ctx, symbols, symbolStatus, false)
}

func (e *Exchange) getLatestSpotPrice(ctx context.Context, symbols currency.Pairs, symbolStatus SymbolStatus, useWebsocket bool) ([]SymbolPriceResponse, error) {
	params := url.Values{}
	if err := e.setSymbolParams(params, symbols); err != nil {
		return nil, err
	}
	if symbolStatus != "" {
		params.Set("symbolStatus", string(symbolStatus))
	}
	var resp objectOrArray[SymbolPriceResponse]
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/ticker/price", "ticker.price", params, symbolsTickerLimit(len(symbols)), &resp)
}

// GetAveragePrice returns the current average price for a symbol
func (e *Exchange) GetAveragePrice(ctx context.Context, symbol currency.Pair) (*AveragePriceResponse, error) {
	return e.getAveragePrice(ctx, symbol, false)
}

func (e *Exchange) getAveragePrice(ctx context.Context, symbol currency.Pair, useWebsocket bool) (*AveragePriceResponse, error) {
	symbolValue, err := e.formatSymbol(symbol)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *AveragePriceResponse
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/avgPrice", "avgPrice", params, averagePriceRate, &resp)
}

// GetBestPrice returns the symbol order book ticker: the best bid and ask price and quantity of each symbol, or of
// every symbol when none are given
func (e *Exchange) GetBestPrice(ctx context.Context, symbols currency.Pairs, symbolStatus SymbolStatus) ([]BookTickerResponse, error) {
	return e.getBestPrice(ctx, symbols, symbolStatus, false)
}

func (e *Exchange) getBestPrice(ctx context.Context, symbols currency.Pairs, symbolStatus SymbolStatus, useWebsocket bool) ([]BookTickerResponse, error) {
	params := url.Values{}
	if err := e.setSymbolParams(params, symbols); err != nil {
		return nil, err
	}
	if symbolStatus != "" {
		params.Set("symbolStatus", string(symbolStatus))
	}
	var resp objectOrArray[BookTickerResponse]
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/ticker/bookTicker", "ticker.book", params, symbolsTickerLimit(len(symbols)), &resp)
}

// GetPriceChangeStats returns 24hr ticker price change statistics. A request without symbols returns every symbol, at
// a much higher weight
func (e *Exchange) GetPriceChangeStats(ctx context.Context, req *PriceChangeStatsRequest) ([]PriceChangeStatsResponse, error) {
	return e.getPriceChangeStats(ctx, req, false)
}

func (e *Exchange) getPriceChangeStats(ctx context.Context, req *PriceChangeStatsRequest, useWebsocket bool) ([]PriceChangeStatsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := e.setSymbolParams(params, req.Symbols); err != nil {
		return nil, err
	}
	if req.Type != "" {
		params.Set("type", string(req.Type))
	}
	if req.SymbolStatus != "" {
		params.Set("symbolStatus", string(req.SymbolStatus))
	}
	var resp objectOrArray[PriceChangeStatsResponse]
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/ticker/24hr", "ticker.24hr", params, ticker24HourLimit(len(req.Symbols)), &resp)
}

// GetTickerData returns rolling window price change statistics of up to 100 symbols. The window's open time starts on a
// minute while its close time is the time of the request, so the window can be up to 59999ms wider than the requested
// WindowSize
func (e *Exchange) GetTickerData(ctx context.Context, req *RollingWindowTickerRequest) ([]TickerStatisticsResponse, error) {
	return e.getTickerData(ctx, req, false)
}

func (e *Exchange) getTickerData(ctx context.Context, req *RollingWindowTickerRequest, useWebsocket bool) ([]TickerStatisticsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.Symbols) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	if err := checkTickerSymbols(len(req.Symbols)); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := e.setSymbolParams(params, req.Symbols); err != nil {
		return nil, err
	}
	if req.WindowSize != 0 {
		windowSize, err := windowSizeToString(req.WindowSize)
		if err != nil {
			return nil, err
		}
		params.Set("windowSize", windowSize)
	}
	if req.Type != "" {
		params.Set("type", string(req.Type))
	}
	if req.SymbolStatus != "" {
		params.Set("symbolStatus", string(req.SymbolStatus))
	}
	var resp objectOrArray[TickerStatisticsResponse]
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/ticker", "ticker", params, rollingWindowTickerLimit(len(req.Symbols)), &resp)
}

// GetTradingDayTicker returns the price change statistics of the current trading day for each of up to 100 symbols
func (e *Exchange) GetTradingDayTicker(ctx context.Context, req *TradingDayTickerRequest) ([]TickerStatisticsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.Symbols) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	if err := checkTickerSymbols(len(req.Symbols)); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := e.setSymbolParams(params, req.Symbols); err != nil {
		return nil, err
	}
	if req.TimeZone != "" {
		params.Set("timeZone", req.TimeZone)
	}
	if req.Type != "" {
		params.Set("type", string(req.Type))
	}
	if req.SymbolStatus != "" {
		params.Set("symbolStatus", string(req.SymbolStatus))
	}
	var resp objectOrArray[TickerStatisticsResponse]
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/ticker/tradingDay", params), tradingDayTickerLimit(len(req.Symbols)), &resp)
}

// windowSizeToString formats a rolling window as a whole number of minutes, hours or days, the units the rolling window
// ticker accepts
func windowSizeToString(windowSize time.Duration) (string, error) {
	const day = 24 * time.Hour
	switch {
	case windowSize >= day && windowSize <= 7*day && windowSize%day == 0:
		return strconv.FormatInt(int64(windowSize/day), 10) + "d", nil
	case windowSize >= time.Hour && windowSize < day && windowSize%time.Hour == 0:
		return strconv.FormatInt(int64(windowSize/time.Hour), 10) + "h", nil
	case windowSize >= time.Minute && windowSize < time.Hour && windowSize%time.Minute == 0:
		return strconv.FormatInt(int64(windowSize/time.Minute), 10) + "m", nil
	}
	return "", fmt.Errorf("%w: %s", errInvalidWindowSize, windowSize)
}

// klineInterval pairs a kline interval with the text Binance.US names it by
type klineInterval struct {
	String   string
	Interval kline.Interval
}

// klineIntervalList holds every kline interval Binance.US serves; it has no one second klines
var klineIntervalList = []klineInterval{
	{String: "1m", Interval: kline.OneMin},
	{String: "3m", Interval: kline.ThreeMin},
	{String: "5m", Interval: kline.FiveMin},
	{String: "15m", Interval: kline.FifteenMin},
	{String: "30m", Interval: kline.ThirtyMin},
	{String: "1h", Interval: kline.OneHour},
	{String: "2h", Interval: kline.TwoHour},
	{String: "4h", Interval: kline.FourHour},
	{String: "6h", Interval: kline.SixHour},
	{String: "8h", Interval: kline.EightHour},
	{String: "12h", Interval: kline.TwelveHour},
	{String: "1d", Interval: kline.OneDay},
	{String: "3d", Interval: kline.ThreeDay},
	{String: "1w", Interval: kline.OneWeek},
	{String: "1M", Interval: kline.OneMonth},
}

// formatToInterval returns the interval of a kline interval string
func formatToInterval(intervalString string) (kline.Interval, error) {
	for _, interval := range klineIntervalList {
		if interval.String == intervalString {
			return interval.Interval, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", kline.ErrInvalidInterval, intervalString)
}

// intervalToString returns the kline interval string of an interval
func intervalToString(interval kline.Interval) (string, error) {
	for _, i := range klineIntervalList {
		if i.Interval == interval {
			return i.String, nil
		}
	}
	return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
}
