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
	ErrUnsetName = errors.New("unset exchange name")
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
	return &Builder{name, pair, formatted, a, request, required, start, end}, nil
}

// GetRanges returns the date ranges for candle intervals broken up over
// requests
func (b *Builder) GetRanges(limit uint32) (*IntervalRangeHolder, error) {
	return CalculateCandleDateRanges(b.Start, b.End, b.Request, limit)
}

// ConvertCandles converts time series candles into a kline.Item type. This will
// auto convert from a lower to higher time series if applicable.
func (b *Builder) ConvertCandles(timeSeries []Candle) (*Item, error) {
	holder := &Item{
		Exchange: b.Name,
		Pair:     b.Pair,
		Asset:    b.Asset,
		Interval: b.Request,
		Candles:  timeSeries,
	}

	holder.RemoveDuplicates()
	holder.RemoveOutsideRange(b.Start, b.End)
	holder.SortCandlesByTimestamp(false)

	if b.Required == b.Request {
		return holder, nil
	}
	// TODO: Fix
	return ConvertToNewInterval(holder, b.Required)
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
	holder, err := b.Builder.ConvertCandles(timeSeries)
	if err != nil {
		return nil, err
	}

	b.SetHasDataFromCandles(holder.Candles)
	summary := b.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", b.Name, summary)
	}
	return holder, nil
}
