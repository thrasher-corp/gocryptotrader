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
	errNilBuilder = errors.New("nil kline builder")
)

// Builder is a helper to request and convert time series to a required candle
// interval.
type Builder struct {
	Name      string
	Pair      currency.Pair
	Formatted currency.Pair
	Asset     asset.Item
	Request   Interval
	Required  Interval
	Start     time.Time
	End       time.Time
}

// GetBuilder generates a builder for interval conversions supported by an
// exchange.
func GetBuilder(name string, pair, formatted currency.Pair, a asset.Item, required, request Interval, start, end time.Time) (*Builder, error) {
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
	if request == 0 {
		return nil, fmt.Errorf("request %w", ErrUnsetInterval)
	}
	err := common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}
	// Force alignment to request interval
	start = start.Truncate(request.Duration())
	end = end.Truncate(request.Duration())
	return &Builder{name, pair, formatted, a, request, required, start, end}, nil
}

// GetRanges returns the date ranges for candle intervals broken up over
// requests
func (b *Builder) GetRanges(limit uint32) (*IntervalRangeHolder, error) {
	if b == nil {
		return nil, errNilBuilder
	}
	return CalculateCandleDateRanges(b.Start, b.End, b.Request, limit)
}

// ConvertCandles converts time series candles into a kline.Item type. This will
// auto convert from a lower to higher time series if applicable.
func (b *Builder) ConvertCandles(timeSeries []Candle) (*Item, error) {
	if b == nil {
		return nil, errNilBuilder
	}

	holder := &Item{
		Exchange: b.Name,
		Pair:     b.Pair,
		Asset:    b.Asset,
		Interval: b.Request,
		Candles:  timeSeries,
	}

	// NOTE: timeSeries param above must keep underlying slice reference in this
	// function as it is used for method ConvertCandles on type BuilderExtended
	// for SetHasDataFromCandles candle matching.
	// TODO: Shift burden of proof to the caller e.g. only find duplicates and error.
	holder.RemoveDuplicates()
	holder.RemoveOutsideRange(b.Start, b.End)
	holder.SortCandlesByTimestamp(false)

	if b.Required == b.Request {
		return holder, nil
	}
	return holder.ConvertToNewInterval(b.Required)
}

// BuilderExtended used in extended functionality for when candles requested
// exceed exchange limits and require multiple requests.
type BuilderExtended struct {
	*Builder
	*IntervalRangeHolder
}

// ConvertCandles converts time series candles into a kline.Item type. This will
// auto convert from a lower to higher time series if applicable.
func (b *BuilderExtended) ConvertCandles(timeSeries []Candle) (*Item, error) {
	if b == nil {
		return nil, errNilBuilder
	}

	holder, err := b.Builder.ConvertCandles(timeSeries)
	if err != nil {
		return nil, err
	}

	// This checks from pre-converted time series data for date range matching.
	// NOTE: If there are any optimizations which copy timeSeries param slice
	// in the function call ConvertCandles above then false positives can
	// occur. // TODO: Improve implementation.
	b.SetHasDataFromCandles(timeSeries)
	summary := b.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", b.Name, summary)
	}
	return holder, nil
}

// Size returns the max length of return for pre-allocation.
func (b *BuilderExtended) Size() int {
	if b == nil || b.IntervalRangeHolder == nil {
		return 0
	}
	if b.IntervalRangeHolder.Limit == 0 {
		log.Warnf(log.ExchangeSys, "%v candle builder limit is zero while calling Size()", b.Name)
	}
	return b.IntervalRangeHolder.Limit * len(b.IntervalRangeHolder.Ranges)
}
