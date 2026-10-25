package cryptodotcom

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	errValuationTypeRequired   = errors.New("valuation type required")
	errInstrumentTypeRequired  = errors.New("instrument type required")
	errMalformedOrderbookLevel = errors.New("malformed order book level")
)

// GetRiskParameters returns Smart Cross Margin's risk parameters
func (e *Exchange) GetRiskParameters(ctx context.Context) (*RiskParametersResponse, error) {
	var resp *RiskParametersResponse
	if err := e.SendHTTPRequest(ctx, publicEPL, "public/get-risk-parameters", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetInstruments returns every instrument Crypto.com trades. The docs' event and combo filters serve prediction
// markets, which this exchange does not list, so it ignores them and sends every instrument in one response
func (e *Exchange) GetInstruments(ctx context.Context) ([]InstrumentResponse, error) {
	var resp *dataResponse[InstrumentResponse]
	if err := e.SendHTTPRequest(ctx, publicEPL, "public/get-instruments", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetOrderbook returns an instrument's order book
func (e *Exchange) GetOrderbook(ctx context.Context, req *OrderbookRequest) (*OrderbookResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	params := url.Values{}
	params.Set("instrument_name", req.InstrumentName)
	if req.Depth > 0 {
		params.Set("depth", strconv.FormatUint(req.Depth, 10))
	}
	var resp *OrderbookResponse
	if err := e.SendHTTPRequest(ctx, publicEPL, "public/get-book", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetCandlestick returns an instrument's candles, oldest first: the latest Count whose periods overlap the window
// from StartTime to EndTime, so a StartTime within a period includes that period's candle
func (e *Exchange) GetCandlestick(ctx context.Context, req *CandlestickRequest) (*CandlestickResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	params := url.Values{}
	params.Set("instrument_name", req.InstrumentName)
	if req.Interval != 0 {
		timeframe, err := intervalToTimeframe(req.Interval)
		if err != nil {
			return nil, err
		}
		params.Set("timeframe", timeframe)
	}
	if req.Count > 0 {
		params.Set("count", strconv.FormatUint(req.Count, 10))
	}
	// Crypto.com reads candle bounds as milliseconds whatever their length, so a nanosecond bound finds nothing
	if err := setTimeWindowParams(params, req.StartTime, req.EndTime, unixMilliParam); err != nil {
		return nil, err
	}
	var resp *CandlestickResponse
	if err := e.SendHTTPRequest(ctx, publicEPL, "public/get-candlestick", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTrades returns an instrument's public trades, newest first: the latest Count from StartTime, inclusive, to
// EndTime, exclusive. Crypto.com refuses an EndTime more than a minute after its clock
func (e *Exchange) GetTrades(ctx context.Context, req *TradesRequest) ([]TradeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	params := url.Values{}
	params.Set("instrument_name", req.InstrumentName)
	if req.Count > 0 {
		params.Set("count", strconv.FormatUint(req.Count, 10))
	}
	if err := setTimeWindowParams(params, req.StartTime, req.EndTime, unixNanoParam); err != nil {
		return nil, err
	}
	var resp *dataResponse[TradeResponse]
	if err := e.SendHTTPRequest(ctx, publicEPL, "public/get-trades", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetTickers returns the ticker of the instrument requested, or of every instrument
func (e *Exchange) GetTickers(ctx context.Context, req *TickersRequest) ([]TickerResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var params url.Values
	if req.InstrumentName != "" {
		params = url.Values{}
		params.Set("instrument_name", req.InstrumentName)
	}
	var resp *dataResponse[TickerResponse]
	if err := e.SendHTTPRequest(ctx, publicEPL, "public/get-tickers", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetValuations returns an instrument's values of a valuation type, newest first: the latest Count before EndTime.
// Crypto.com serves funding history from at most 14 days before EndTime, and can include the rate of the hour before
// StartTime
func (e *Exchange) GetValuations(ctx context.Context, req *ValuationsRequest) (*ValuationsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if req.ValuationType == "" {
		return nil, errValuationTypeRequired
	}
	params := url.Values{}
	params.Set("instrument_name", req.InstrumentName)
	params.Set("valuation_type", req.ValuationType)
	if req.Count > 0 {
		params.Set("count", strconv.FormatUint(req.Count, 10))
	}
	if err := setTimeWindowParams(params, req.StartTime, req.EndTime, unixMilliParam); err != nil {
		return nil, err
	}
	var resp *ValuationsResponse
	if err := e.SendHTTPRequest(ctx, publicEPL, "public/get-valuations", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetExpiredSettlementPrice returns the prices expired instruments of a type settled at, the latest expiries first
func (e *Exchange) GetExpiredSettlementPrice(ctx context.Context, req *ExpiredSettlementPriceRequest) ([]ExpiredSettlementPriceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	params.Set("instrument_type", req.InstrumentType)
	if req.Page > 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	var resp *dataResponse[ExpiredSettlementPriceResponse]
	if err := e.SendHTTPRequest(ctx, publicEPL, "public/get-expired-settlement-price", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetDepositInstruments returns the currencies and tokens that can be deposited, and whether each counts as collateral
func (e *Exchange) GetDepositInstruments(ctx context.Context) ([]DepositInstrumentResponse, error) {
	var resp []DepositInstrumentResponse
	if err := e.SendHTTPRequest(ctx, publicEPL, "public/get-deposit-instruments", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetInsurance returns a currency's insurance fund balances, hourly and newest first: the latest Count before EndTime,
// from at most 14 days before it
func (e *Exchange) GetInsurance(ctx context.Context, req *InsuranceRequest) (*InsuranceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("instrument_name", req.Currency.Upper().String())
	if req.Count > 0 {
		params.Set("count", strconv.FormatUint(req.Count, 10))
	}
	if err := setTimeWindowParams(params, req.StartTime, req.EndTime, unixMilliParam); err != nil {
		return nil, err
	}
	var resp *InsuranceResponse
	if err := e.SendHTTPRequest(ctx, publicEPL, "public/get-insurance", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// setTimeWindowParams sets the start_ts and end_ts of a time window, each converted by toParam and left out when unset
func setTimeWindowParams(params url.Values, start, end time.Time, toParam func(time.Time) (uint64, error)) error {
	if !start.IsZero() && !end.IsZero() {
		if err := common.StartEndTimeCheck(start, end); err != nil {
			return err
		}
	}
	startTS, err := toParam(start)
	if err != nil {
		return err
	}
	endTS, err := toParam(end)
	if err != nil {
		return err
	}
	if startTS > 0 {
		params.Set("start_ts", strconv.FormatUint(startTS, 10))
	}
	if endTS > 0 {
		params.Set("end_ts", strconv.FormatUint(endTS, 10))
	}
	return nil
}

// intervalToTimeframe returns the candlestick timeframe Crypto.com names an interval by
func intervalToTimeframe(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1m", nil
	case kline.FiveMin:
		return "5m", nil
	case kline.FifteenMin:
		return "15m", nil
	case kline.ThirtyMin:
		return "30m", nil
	case kline.OneHour:
		return "1h", nil
	case kline.TwoHour:
		return "2h", nil
	case kline.FourHour:
		return "4h", nil
	case kline.TwelveHour:
		return "12h", nil
	case kline.OneDay:
		return "1D", nil
	case kline.SevenDay:
		return "7D", nil
	case kline.TwoWeek:
		return "14D", nil
	case kline.OneMonth:
		return "1M", nil
	default:
		return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
	}
}

// timeframeToInterval returns the interval of a candlestick timeframe, as Crypto.com names it in a response or a
// candlestick channel
func timeframeToInterval(timeframe string) (kline.Interval, error) {
	switch timeframe {
	case "1m":
		return kline.OneMin, nil
	case "5m":
		return kline.FiveMin, nil
	case "15m":
		return kline.FifteenMin, nil
	case "30m":
		return kline.ThirtyMin, nil
	case "1h":
		return kline.OneHour, nil
	case "2h":
		return kline.TwoHour, nil
	case "4h":
		return kline.FourHour, nil
	case "12h":
		return kline.TwelveHour, nil
	case "1D":
		return kline.OneDay, nil
	case "7D":
		return kline.SevenDay, nil
	case "14D":
		return kline.TwoWeek, nil
	case "1M":
		return kline.OneMonth, nil
	default:
		return 0, fmt.Errorf("%w: %q", kline.ErrInvalidInterval, timeframe)
	}
}

// UnmarshalJSON decodes a level from its [price, quantity, number of orders] array, whose values Crypto.com quotes
func (l *OrderbookLevel) UnmarshalJSON(data []byte) error {
	var fields []json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("%w: %w", errMalformedOrderbookLevel, err)
	}
	if len(fields) != 3 {
		return fmt.Errorf("%w: %d values, not a price, quantity and number of orders", errMalformedOrderbookLevel, len(fields))
	}
	if err := l.Price.UnmarshalJSON(fields[0]); err != nil {
		return fmt.Errorf("%w: price: %w", errMalformedOrderbookLevel, err)
	}
	if err := l.Quantity.UnmarshalJSON(fields[1]); err != nil {
		return fmt.Errorf("%w: quantity: %w", errMalformedOrderbookLevel, err)
	}
	orderCount, err := strconv.ParseUint(strings.Trim(string(fields[2]), `"`), 10, 64)
	if err != nil {
		return fmt.Errorf("%w: number of orders: %w", errMalformedOrderbookLevel, err)
	}
	l.OrderCount = orderCount
	return nil
}
