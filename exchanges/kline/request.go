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
	ErrUnsetName = errors.New("unset exchange name")
	// ErrNoTimeSeriesDataToConvert is returned when no data can be processed
	ErrNoTimeSeriesDataToConvert = errors.New("no candle data returned to process")

	errNilRequest                   = errors.New("nil kline request")
	errInvalidSpecificEndpointLimit = errors.New("specific endpoint limit must be greater than 0")

	// PartialCandle is string flag for when the most recent candle is partially
	// formed.
	PartialCandle = "Partial Candle"
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
	// ClientRequired refers to the clients' actual required interval
	// needed.
	ClientRequired Interval
	// Start is the start time aligned to UTC and to the Required interval candle
	Start time.Time
	// End is the end time aligned to UTC and to the Required interval candle
	End time.Time
	// PartialCandle defines when a request's end time interval goes beyond
	// current time it potentially has a partially formed candle.
	PartialCandle bool
	// IsExtended denotes whether the candle request is for extended candles
	IsExtended bool
	// ProcessedCandles stores the candles that have been processed, but not converted
	// to the ClientRequiredInterval
	ProcessedCandles []Candle
	// RequestLimit is the potential maximum amount of candles that can be
	// returned
	RequestLimit uint64
}

// CreateKlineRequest generates a `Request` type for interval conversions
// supported by an exchange.
func CreateKlineRequest(name string, pair, formatted currency.Pair, a asset.Item, clientRequired, exchangeInterval Interval, start, end time.Time, specificEndpointLimit uint64) (*Request, error) {
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

	if specificEndpointLimit <= 0 {
		return nil, errInvalidSpecificEndpointLimit
	}

	// Force UTC alignment
	start = start.UTC()
	end = end.UTC()

	// Force alignment to required interval which is the higher time value e.g.
	// 1hr required as opposed to 1min request/outbound interval used to
	// construct the higher time value candle. This is to make sure there are
	// minimal missing candles which is used to create the bigger candle.
	start = start.Truncate(clientRequired.Duration())

	// Strip future time to current time so there is no extra padding.
	if end.After(time.Now()) {
		end = time.Now().UTC()
	}

	// Strip monotonic clock reading for comparison
	end = end.Round(0)

	endTrunc := end.Truncate(clientRequired.Duration())
	// Check to see if truncation moves end time and if so we want to make sure
	// the candle period is included on the end.
	if !endTrunc.Equal(end) {
		end = endTrunc.Add(clientRequired.Duration())
	}

	return &Request{
		Exchange:         name,
		Pair:             pair,
		RequestFormatted: formatted,
		Asset:            a,
		ExchangeInterval: exchangeInterval,
		ClientRequired:   clientRequired,
		Start:            start,
		End:              end,
		PartialCandle:    end.After(time.Now()),
		RequestLimit:     specificEndpointLimit,
	}, nil
}

// GetRanges returns the date ranges for candle intervals broken up over
// requests
func (r *Request) GetRanges(limit uint64) (*IntervalRangeHolder, error) {
	if r == nil {
		return nil, errNilRequest
	}
	return CalculateCandleDateRanges(r.Start, r.End, r.ExchangeInterval, limit)
}

// ProcessResponse converts time series candles into a kline.Item type. This
// will auto convert from a lower to higher time series if applicable.
func (r *Request) ProcessResponse(timeSeries []Candle) (*Item, error) {
	if r == nil {
		return nil, errNilRequest
	}

	if len(timeSeries) == 0 {
		return nil, ErrNoTimeSeriesDataToConvert
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
	err := holder.addPadding(r.Start, r.End, r.PartialCandle)
	if err != nil {
		return nil, err
	}

	if r.IsExtended {
		// NOTE: This allows for a processed candles to be analysed
		// in the context of ExtendedRequest's ProcessResponse function
		r.ProcessedCandles = make([]Candle, len(holder.Candles))
		copy(r.ProcessedCandles, holder.Candles)
	}
	if r.ClientRequired != r.ExchangeInterval {
		holder, err = holder.ConvertToNewInterval(r.ClientRequired)
	}

	if r.PartialCandle {
		// NOTE: Some endpoints do not return incomplete candles, verify for
		// incomplete candle.
		recentCandle := &holder.Candles[len(holder.Candles)-1]
		if recentCandle.Time.Add(r.ClientRequired.Duration()).After(time.Now()) {
			recentCandle.ValidationIssues = PartialCandle
		}
	}

	return holder, err
}

// Size returns the max length of return for pre-allocation.
func (r *Request) Size() uint64 {
	if r == nil {
		return 0
	}

	return TotalCandlesPerInterval(r.Start, r.End, r.ExchangeInterval)
}

// ExtendedRequest used in extended functionality for when candles requested
// exceed exchange limits and require multiple requests.
type ExtendedRequest struct {
	*Request
	RangeHolder *IntervalRangeHolder
}

// ProcessResponse converts time series candles into a kline.Item type. This
// will auto convert from a lower to higher time series if applicable.
func (r *ExtendedRequest) ProcessResponse(timeSeries []Candle) (*Item, error) {
	if r == nil {
		return nil, errNilRequest
	}

	if len(timeSeries) == 0 {
		return nil, ErrNoTimeSeriesDataToConvert
	}

	holder, err := r.Request.ProcessResponse(timeSeries)
	if err != nil {
		return nil, err
	}
	err = r.RangeHolder.SetHasDataFromCandles(r.Request.ProcessedCandles)
	if err != nil {
		return nil, err
	}

	summary := r.RangeHolder.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", r.Exchange, summary)
	}
	return holder, nil
}

// Size returns the max length of return for pre-allocation.
func (r *ExtendedRequest) Size() uint64 {
	if r == nil || r.RangeHolder == nil {
		return 0
	}
	if r.RangeHolder.Limit == 0 {
		log.Warnf(log.ExchangeSys, "%v candle request limit is zero while calling Size()", r.Exchange)
	}
	return r.RangeHolder.Limit * uint64(len(r.RangeHolder.Ranges))
}
