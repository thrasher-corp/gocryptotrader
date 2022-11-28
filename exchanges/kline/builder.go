package kline

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var ErrNilBuilder = errors.New("nil kline builder")

// Builder is a helper to request and convert time series to a required candle
// interval.
type Builder struct {
	Name     string
	Pair     currency.Pair
	Asset    asset.Item
	Request  Interval
	Required Interval
	Start    time.Time
	End      time.Time
}

var ErrUnsetName = errors.New("unset exchange name")

// GetBuilder generates a builder for interval conversions supported by an
// exchange. Request
func GetBuilder(name string, pair currency.Pair, a asset.Item, required, request Interval, start, end time.Time) (*Builder, error) {
	if name == "" {
		return nil, ErrUnsetName
	}
	if pair.IsEmpty() {
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
	return &Builder{name, pair, a, request, required, start, end}, nil
}

// GetRanges returns the date ranges for candle intervals broken up over
// requests
func (b *Builder) GetRanges(limit uint32) (*IntervalRangeHolder, error) {
	return CalculateCandleDateRanges(b.Start, b.End, b.Request, limit)
}

// ConvertCandles converts time series candles into a kline.Item type. This will
// auto convert from a lower to higher time series if applicable.
func (b *Builder) ConvertCandles(timeSeries []Candle) (*Item, error) {
	if b.Required == b.Request {
		return &Item{
			Exchange: b.Name,
			Pair:     b.Pair,
			Asset:    b.Asset,
			Interval: b.Required,
			Candles:  timeSeries,
		}, nil
	}

	return ConvertToNewInterval(&Item{
		Exchange: b.Name,
		Pair:     b.Pair,
		Asset:    b.Asset,
		Interval: b.Request,
		Candles:  timeSeries,
	}, b.Required)
}

// Convert takes in candles from a lower order time series to be converted to
// a higher time series.
func (b *Builder) Convert(incoming *Item) (*Item, error) {
	return ConvertToNewInterval(incoming, b.Required)
}
