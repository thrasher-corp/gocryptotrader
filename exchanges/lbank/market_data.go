package lbank

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const (
	maxDepthSize  = 200
	maxTradesSize = 500
	maxKlineSize  = 2000
)

// klineTypes holds the candle intervals LBank serves. Its docs also list hour8 and hour12, which LBank refuses with an
// internal error
var klineTypes = map[kline.Interval]string{
	kline.OneMin:     "minute1",
	kline.FiveMin:    "minute5",
	kline.FifteenMin: "minute15",
	kline.ThirtyMin:  "minute30",
	kline.OneHour:    "hour1",
	kline.FourHour:   "hour4",
	kline.OneDay:     "day1",
	kline.OneWeek:    "week1",
	kline.OneMonth:   "month1",
}

// Ping checks the REST API can be reached
func (e *Exchange) Ping(ctx context.Context) error {
	return e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodPost, "/v2/supplement/system_ping.do", nil, nil)
}

// GetMarketDepth returns a pair's order book to the depth given, of 1 to 200 levels a side
func (e *Exchange) GetMarketDepth(ctx context.Context, pair currency.Pair, size uint64) (*MarketDepthResponse, error) {
	if size == 0 || size > maxDepthSize {
		return nil, fmt.Errorf("%w: size %d must be 1 to %d", errLimitOutOfRange, size, maxDepthSize)
	}
	params, err := e.symbolParams(pair)
	if err != nil {
		return nil, err
	}
	params.Set("size", strconv.FormatUint(size, 10))
	var resp *MarketDepthResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodGet, "/v2/depth.do", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLatestPrices returns a pair's latest price, or every pair's when the pair is empty
func (e *Exchange) GetLatestPrices(ctx context.Context, pair currency.Pair) ([]LatestPriceResponse, error) {
	params := url.Values{}
	if !pair.IsEmpty() {
		var err error
		if params, err = e.symbolParams(pair); err != nil {
			return nil, err
		}
	}
	var resp []LatestPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodGet, "/v2/supplement/ticker/price.do", params, &resp)
}

// GetBookTicker returns a pair's best bid and ask
func (e *Exchange) GetBookTicker(ctx context.Context, pair currency.Pair) (*BookTickerResponse, error) {
	params, err := e.symbolParams(pair)
	if err != nil {
		return nil, err
	}
	var resp *BookTickerResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodGet, "/v2/supplement/ticker/bookTicker.do", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTickers returns a pair's 24 hour statistics, or every pair's when the pair is empty. Leveraged tokens are left
// out; GetLeveragedTokenTickers returns them
func (e *Exchange) GetTickers(ctx context.Context, pair currency.Pair) ([]TickerResponse, error) {
	return e.getTickers(ctx, "/v2/ticker/24hr.do", pair)
}

// GetLeveragedTokenTickers returns a leveraged token's 24 hour statistics, or every leveraged token's when the pair is
// empty
func (e *Exchange) GetLeveragedTokenTickers(ctx context.Context, pair currency.Pair) ([]TickerResponse, error) {
	return e.getTickers(ctx, "/v2/etfTicker/24hr.do", pair)
}

func (e *Exchange) getTickers(ctx context.Context, path string, pair currency.Pair) ([]TickerResponse, error) {
	params := url.Values{}
	params.Set("symbol", "all")
	if !pair.IsEmpty() {
		var err error
		if params, err = e.symbolParams(pair); err != nil {
			return nil, err
		}
	}
	var resp []TickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodGet, path, params, &resp)
}

// GetTrades returns up to size trades of a pair, 1 to 500. With a zero time it returns the latest trades, newest first;
// otherwise it returns the trades after that time, oldest first
func (e *Exchange) GetTrades(ctx context.Context, pair currency.Pair, size uint64, after time.Time) ([]TradeResponse, error) {
	if size == 0 || size > maxTradesSize {
		return nil, fmt.Errorf("%w: size %d must be 1 to %d", errLimitOutOfRange, size, maxTradesSize)
	}
	params, err := e.symbolParams(pair)
	if err != nil {
		return nil, err
	}
	params.Set("size", strconv.FormatUint(size, 10))
	if !after.IsZero() {
		params.Set("time", strconv.FormatInt(after.UnixMilli(), 10))
	}
	var resp []TradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodGet, "/v2/supplement/trades.do", params, &resp)
}

// GetKlines returns up to Size candles of a pair, 1 to 2000, starting with the candle holding the start time.
// LBank's day, week and month candles start at midnight UTC+8
func (e *Exchange) GetKlines(ctx context.Context, arg *KlineRequest) ([]KlineResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Size == 0 || arg.Size > maxKlineSize {
		return nil, fmt.Errorf("%w: size %d must be 1 to %d", errLimitOutOfRange, arg.Size, maxKlineSize)
	}
	klineType, ok := klineTypes[arg.Interval]
	if !ok {
		return nil, fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, arg.Interval)
	}
	if arg.Start.IsZero() {
		return nil, fmt.Errorf("start %w", common.ErrDateUnset)
	}
	params, err := e.symbolParams(arg.Pair)
	if err != nil {
		return nil, err
	}
	params.Set("size", strconv.FormatUint(arg.Size, 10))
	params.Set("type", klineType)
	params.Set("time", strconv.FormatInt(arg.Start.Unix(), 10))
	var resp []KlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodGet, "/v2/kline.do", params, &resp)
}

// symbolParams returns parameters holding a spot pair's symbol
func (e *Exchange) symbolParams(pair currency.Pair) (url.Values, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbol, err := e.FormatSymbol(pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	return params, nil
}
