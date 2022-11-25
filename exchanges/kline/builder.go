package kline

import "fmt"

// Builder is a helper to request and convert time series to a required candle
// interval.
type Builder struct {
	request  Interval
	required Interval
}

// GetBuilder generates a builder for interval conversions supported by an
// exchange. Request
func GetBuilder(request, required Interval) (*Builder, error) {
	if request == 0 {
		return nil, fmt.Errorf("request %w", ErrUnsetInterval)
	}
	if required == 0 {
		return nil, fmt.Errorf("required %w", ErrUnsetInterval)
	}
	return &Builder{request, required}, nil
}

// Request returns the interval supported by the exchange which can then be
// used to build higher time series candles.
func (b *Builder) Request() Interval {
	return b.request
}

// Required returns the interval that is required to be converted to.
func (b *Builder) Required() Interval {
	return b.required
}

// Convert takes in candles from a lower order time series to be converted to
// a higher time series.
func (b *Builder) Convert(incoming *Item) (*Item, error) {
	return ConvertToNewInterval(incoming, b.required)
}
