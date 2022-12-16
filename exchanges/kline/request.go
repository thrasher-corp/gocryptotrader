package kline

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	// ErrUnsetName is an error for when the exchange name is not set
	ErrUnsetName  = errors.New("unset exchange name")
	errNilRequest = errors.New("nil kline request")
)

// Request is a helper to request and convert time series to a required candle
// interval.
type Request struct {
	// Name refers to the exchange name
	Name string
	// Pair refers to the currency pair
	Pair currency.Pair
	// Formatted refers to the currency pair formatted by the exchange asset
	// for outbound requests
	Formatted currency.Pair
	// Asset refers to the asset type
	Asset asset.Item
	// Outbound refers to the interval that is used to construct the required
	// interval this will be less than or equal to the required interval.
	Outbound Interval
	// Required refers to the actual required interval needed
	Required Interval
	// Start is the start time aligned to UTC and to the Required interval candle
	Start time.Time
	// End is the end time aligned to UTC and to the Required interval candle
	End time.Time
}

// CreateKlineRequest generates a `Request` type for interval conversions
// supported by an exchange.
func CreateKlineRequest(name string, pair, formatted currency.Pair, a asset.Item, required, outbound Interval, start, end time.Time) (*Request, error) {
	if name == "" {
		return nil, ErrUnsetName
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if formatted.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, asset.ErrNotSupported
	}
	if required == 0 {
		return nil, fmt.Errorf("required %w", ErrUnsetInterval)
	}
	if outbound == 0 {
		return nil, fmt.Errorf("request %w", ErrUnsetInterval)
	}
	err := common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}

	// Force UTC alignment
	start = start.UTC()
	end = end.UTC()

	// Force alignment to required interval which is the higher time value e.g.
	// 1hr required as opposed to 1min request/outbound interval used to
	// construct the higher time value candle. This is to make sure there are
	// minimal missing candles which is used to create the bigger candle.
	start = start.Truncate(required.Duration())

	// Strip montonic clock reading for comparison
	end = end.Round(0)

	endTrunc := end.Truncate(required.Duration())
	// Check to see if truncation moves end time and if so we want to make sure
	// the candle period is included on the end.
	forward := endTrunc.Add(required.Duration())
	if !endTrunc.Equal(end) && !forward.After(time.Now()) {
		end = forward
	}
	return &Request{name, pair, formatted, a, outbound, required, start, end}, nil
}

// GetRanges returns the date ranges for candle intervals broken up over
// requests
func (r *Request) GetRanges(limit uint32) (*IntervalRangeHolder, error) {
	if r == nil {
		return nil, errNilRequest
	}
	return CalculateCandleDateRanges(r.Start, r.End, r.Outbound, limit)
}

// ConvertCandles converts time series candles into a kline.Item type. This will
// auto convert from a lower to higher time series if applicable.
func (r *Request) ConvertCandles(timeSeries []Candle) (*Item, error) {
	if r == nil {
		return nil, errNilRequest
	}

	holder := &Item{
		Exchange: r.Name,
		Pair:     r.Pair,
		Asset:    r.Asset,
		Interval: r.Outbound,
		Candles:  timeSeries,
	}

	// NOTE: timeSeries param above must keep underlying slice reference in this
	// function as it is used for method ConvertCandles on type RequestExtended
	// for SetHasDataFromCandles candle matching.
	// TODO: Shift burden of proof to the caller e.g. only find duplicates and error.
	holder.RemoveDuplicates()
	holder.RemoveOutsideRange(r.Start, r.End)
	holder.SortCandlesByTimestamp(false)

	if r.Required == r.Outbound {
		return holder, nil
	}
	return holder.ConvertToNewInterval(r.Required)
}

// RequestExtended used in extended functionality for when candles requested
// exceed exchange limits and require multiple requests.
type RequestExtended struct {
	*Request
	*IntervalRangeHolder
}

// ConvertCandles converts time series candles into a kline.Item type. This will
// auto convert from a lower to higher time series if applicable.
func (r *RequestExtended) ConvertCandles(timeSeries []Candle) (*Item, error) {
	if r == nil {
		return nil, errNilRequest
	}

	holder, err := r.Request.ConvertCandles(timeSeries)
	if err != nil {
		return nil, err
	}

	// This checks from pre-converted time series data for date range matching.
	// NOTE: If there are any optimizations which copy timeSeries param slice
	// in the function call ConvertCandles above then false positives can
	// occur. // TODO: Improve implementation.
	r.SetHasDataFromCandles(timeSeries)
	summary := r.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", r.Name, summary)
	}
	return holder, nil
}

// Size returns the max length of return for pre-allocation.
func (r *RequestExtended) Size() int {
	if r == nil || r.IntervalRangeHolder == nil {
		return 0
	}
	if r.IntervalRangeHolder.Limit == 0 {
		log.Warnf(log.ExchangeSys, "%v candle request limit is zero while calling Size()", r.Name)
	}
	return r.IntervalRangeHolder.Limit * len(r.IntervalRangeHolder.Ranges)
}
