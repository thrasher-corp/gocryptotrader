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
	// Exchange refers to the exchange name
	Exchange string
	// Pair refers to the currency pair
	Pair currency.Pair
	// RequestFormatted refers to the currency pair formatted by the exchange
	// asset for outbound requests
	RequestFormatted currency.Pair
	// Asset refers to the asset type
	Asset asset.Item
	// ExchangeInterval refers to the interval that is used to construct the
	// client required interval, this will be less than or equal to the client
	// required interval.
	ExchangeInterval Interval
	// ClientRequired refers to the actual clients' actual required interval
	// needed.
	ClientRequired Interval
	// Start is the start time aligned to UTC and to the Required interval candle
	Start time.Time
	// End is the end time aligned to UTC and to the Required interval candle
	End time.Time
}

// CreateKlineRequest generates a `Request` type for interval conversions
// supported by an exchange.
func CreateKlineRequest(name string, pair, formatted currency.Pair, a asset.Item, clientRequired, exchangeInterval Interval, start, end time.Time) (*Request, error) {
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
	if clientRequired == 0 {
		return nil, fmt.Errorf("client required %w", ErrInvalidInterval)
	}
	if exchangeInterval == 0 {
		return nil, fmt.Errorf("exchange interval %w", ErrInvalidInterval)
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
	start = start.Truncate(clientRequired.Duration())

	// Strip montonic clock reading for comparison
	end = end.Round(0)

	endTrunc := end.Truncate(clientRequired.Duration())
	// Check to see if truncation moves end time and if so we want to make sure
	// the candle period is included on the end.
	forward := endTrunc.Add(clientRequired.Duration())
	if !endTrunc.Equal(end) && !forward.After(time.Now()) {
		end = forward
	}
	return &Request{name, pair, formatted, a, exchangeInterval, clientRequired, start, end}, nil
}

// GetRanges returns the date ranges for candle intervals broken up over
// requests
func (r *Request) GetRanges(limit uint32) (*IntervalRangeHolder, error) {
	if r == nil {
		return nil, errNilRequest
	}
	return CalculateCandleDateRanges(r.Start, r.End, r.ExchangeInterval, limit)
}

// ConvertCandles converts time series candles into a kline.Item type. This will
// auto convert from a lower to higher time series if applicable.
func (r *Request) ConvertCandles(timeSeries []Candle) (*Item, error) {
	if r == nil {
		return nil, errNilRequest
	}

	holder := &Item{
		Exchange: r.Exchange,
		Pair:     r.Pair,
		Asset:    r.Asset,
		Interval: r.ExchangeInterval,
		Candles:  timeSeries,
	}

	// NOTE: timeSeries param above must keep underlying slice reference in this
	// function as it is used for method ConvertCandles on type ExtendedRequest
	// for SetHasDataFromCandles candle matching.
	// TODO: Shift burden of proof to the caller e.g. only find duplicates and error.
	holder.RemoveDuplicates()
	holder.RemoveOutsideRange(r.Start, r.End)
	holder.SortCandlesByTimestamp(false)

	if r.ClientRequired == r.ExchangeInterval {
		return holder, nil
	}
	return holder.ConvertToNewInterval(r.ClientRequired)
}

// ExtendedRequest used in extended functionality for when candles requested
// exceed exchange limits and require multiple requests.
type ExtendedRequest struct {
	*Request
	*IntervalRangeHolder
}

// ConvertCandles converts time series candles into a kline.Item type. This will
// auto convert from a lower to higher time series if applicable.
func (r *ExtendedRequest) ConvertCandles(timeSeries []Candle) (*Item, error) {
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
		log.Warnf(log.ExchangeSys, "%v - %v", r.Exchange, summary)
	}
	return holder, nil
}

// Size returns the max length of return for pre-allocation.
func (r *ExtendedRequest) Size() int {
	if r == nil || r.IntervalRangeHolder == nil {
		return 0
	}
	if r.IntervalRangeHolder.Limit == 0 {
		log.Warnf(log.ExchangeSys, "%v candle request limit is zero while calling Size()", r.Exchange)
	}
	return r.IntervalRangeHolder.Limit * len(r.IntervalRangeHolder.Ranges)
}
