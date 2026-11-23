package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var errInstrumentIDOrFamilyRequired = errors.New("either instrument ID or instrument family is required")

// GetTickers retrieves the latest price snapshot, best bid and ask, and 24 hour trading volume of every instrument of
// an instrument type
func (e *Exchange) GetTickers(ctx context.Context, arg *TickersRequest) ([]TickerResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	var resp []TickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTickersEPL, http.MethodGet, common.EncodeURLValues("market/tickers", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetTicker retrieves the latest price snapshot, best bid and ask, and 24 hour trading volume of an instrument
func (e *Exchange) GetTicker(ctx context.Context, instrumentID string) (*TickerResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *TickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTickerEPL, http.MethodGet, common.EncodeURLValues("market/ticker", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOrderBookDepth retrieves an instrument's order book
func (e *Exchange) GetOrderBookDepth(ctx context.Context, arg *OrderBookRequest) (*OrderBookResponse, error) {
	params, err := orderBookParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *OrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOrderBookEPL, http.MethodGet, common.EncodeURLValues("market/books", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetRPIOrderBook retrieves an instrument's order book combining organic depth with the currently tradeable retail
// price improvement depth at each price level
func (e *Exchange) GetRPIOrderBook(ctx context.Context, arg *OrderBookRequest) (*RPIOrderBookResponse, error) {
	params, err := orderBookParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *RPIOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getRPIOrderBookEPL, http.MethodGet, common.EncodeURLValues("market/books-rpi", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetFullOrderBook retrieves an instrument's order book up to 5000 levels per side
func (e *Exchange) GetFullOrderBook(ctx context.Context, arg *OrderBookRequest) (*FullOrderBookResponse, error) {
	params, err := orderBookParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *FullOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getFullOrderBookEPL, http.MethodGet, common.EncodeURLValues("market/books-full", params), nil, &resp, request.UnauthenticatedRequest)
}

func orderBookParams(arg *OrderBookRequest) (url.Values, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", arg.InstrumentID)
	if arg.Depth > 0 {
		params.Set("sz", strconv.FormatUint(arg.Depth, 10))
	}
	return params, nil
}

// GetCandlesticks retrieves an instrument's latest 1,440 candlesticks
func (e *Exchange) GetCandlesticks(ctx context.Context, arg *CandlesticksRequest) ([]CandlestickResponse, error) {
	return e.getCandlesticks(ctx, "market/candles", getCandlesticksEPL, arg)
}

// GetCandlesticksHistory retrieves an instrument's candlesticks from recent years
func (e *Exchange) GetCandlesticksHistory(ctx context.Context, arg *CandlesticksRequest) ([]CandlestickResponse, error) {
	return e.getCandlesticks(ctx, "market/history-candles", getCandlestickHistoryEPL, arg)
}

func (e *Exchange) getCandlesticks(ctx context.Context, path string, epl request.EndpointLimit, arg *CandlesticksRequest) ([]CandlestickResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := candlestickParams(arg.InstrumentID, arg.Bar, arg.After, arg.Before, arg.Limit)
	if err != nil {
		return nil, err
	}
	if arg.Adjust != "" {
		params.Set("adjust", arg.Adjust)
	}
	var resp []CandlestickResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, common.EncodeURLValues(path, params), nil, &resp, request.UnauthenticatedRequest)
}

// candlestickParams builds the query shared by the candlestick endpoints, whose after bound returns earlier records
// and before bound newer ones
func candlestickParams(instrumentID string, bar kline.Interval, after, before time.Time, limit uint64) (url.Values, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if !after.IsZero() && !before.IsZero() && before.After(after) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if bar != 0 {
		barString := IntervalFromString(bar, true)
		if barString == "" {
			return nil, fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, bar)
		}
		params.Set("bar", barString)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params, nil
}

// IntervalFromString returns OKX's bar string for an interval; appendUTC selects the UTC opening variant of the
// intervals of six hours and longer, which otherwise open at UTC+8
func IntervalFromString(interval kline.Interval, appendUTC bool) string {
	switch interval {
	case kline.ThousandMilliseconds:
		return "1s"
	case kline.OneMin:
		return "1m"
	case kline.ThreeMin:
		return "3m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1H"
	case kline.TwoHour:
		return "2H"
	case kline.FourHour:
		return "4H"
	case kline.EightHour:
		return "8H"
	}

	duration := ""
	switch interval {
	case kline.SixHour:
		duration = "6H"
	case kline.TwelveHour:
		duration = "12H"
	case kline.OneDay:
		duration = "1D"
	case kline.TwoDay:
		duration = "2D"
	case kline.ThreeDay:
		duration = "3D"
	case kline.FiveDay:
		duration = "5D"
	case kline.OneWeek:
		duration = "1W"
	case kline.OneMonth:
		duration = "1M"
	case kline.ThreeMonth:
		duration = "3M"
	case kline.SixMonth:
		duration = "6M"
	case kline.OneYear:
		duration = "1Y"
	default:
		return duration
	}
	if appendUTC {
		duration += "utc"
	}
	return duration
}

// GetTrades retrieves an instrument's recent trades
func (e *Exchange) GetTrades(ctx context.Context, arg *TradesRequest) ([]TradeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", arg.InstrumentID)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []TradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTradesRequestEPL, http.MethodGet, common.EncodeURLValues("market/trades", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetTradesHistory retrieves an instrument's trades from the last 3 months with pagination
func (e *Exchange) GetTradesHistory(ctx context.Context, arg *TradesHistoryRequest) ([]TradeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", arg.InstrumentID)
	if arg.PaginationType != "" {
		params.Set("type", arg.PaginationType)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []TradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTradesHistoryEPL, http.MethodGet, common.EncodeURLValues("market/history-trades", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOptionTradesByInstrumentFamily retrieves the recent trades of an option instrument family, grouped by option type
func (e *Exchange) GetOptionTradesByInstrumentFamily(ctx context.Context, instrumentFamily string) ([]OptionFamilyTradesResponse, error) {
	if instrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	params := url.Values{}
	params.Set("instFamily", instrumentFamily)
	var resp []OptionFamilyTradesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, optionInstrumentTradeFamilyEPL, http.MethodGet, common.EncodeURLValues("market/option/instrument-family-trades", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOptionTrades retrieves the recent trades of an option instrument or instrument family
func (e *Exchange) GetOptionTrades(ctx context.Context, arg *OptionTradesRequest) ([]OptionTradeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" && arg.InstrumentFamily == "" {
		return nil, errInstrumentIDOrFamilyRequired
	}
	params := url.Values{}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if arg.OptionType != "" {
		params.Set("optType", arg.OptionType)
	}
	var resp []OptionTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, optionTradesEPL, http.MethodGet, common.EncodeURLValues("public/option-trades", params), nil, &resp, request.UnauthenticatedRequest)
}

// Get24HourTotalVolume retrieves the platform's rolling 24 hour order book trading volume
func (e *Exchange) Get24HourTotalVolume(ctx context.Context) (*TotalVolume24HourResponse, error) {
	var resp *TotalVolume24HourResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, get24HTotalVolumeEPL, http.MethodGet, "market/platform-24-volume", nil, &resp, request.UnauthenticatedRequest)
}

// GetCallAuctionDetails retrieves an instrument's call auction details
func (e *Exchange) GetCallAuctionDetails(ctx context.Context, instrumentID string) (*CallAuctionDetailsResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *CallAuctionDetailsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCallAuctionDetailsEPL, http.MethodGet, common.EncodeURLValues("market/call-auction-details", params), nil, &resp, request.UnauthenticatedRequest)
}
