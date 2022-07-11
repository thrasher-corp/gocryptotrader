package kline

import (
	"errors"
	"fmt"
)

var (
	errInvalidElement           = errors.New("invalid element")
	errElementExceedsDataLength = errors.New("element exceeds data length")
	errDataLengthMismatch       = errors.New("data length mismatch")
)

// GetAveragePrice returns the average price from the open, high, low and close
func (c *Candle) GetAveragePrice() float64 {
	return (c.Open + c.High + c.Low + c.Close) / 4
}

// GetAveragePrice returns the average price from the open, high, low and close
func (o *OHLC) GetAveragePrice(element int) (float64, error) {
	if o == nil {
		return 0, fmt.Errorf("get average price %w", errNilOHLC)
	}
	if element < 0 {
		return 0, fmt.Errorf("get average price %w", errInvalidElement)
	}
	check := element + 1
	if check > len(o.Open) || check > len(o.High) || check > len(o.Low) || check > len(o.Close) {
		return 0, fmt.Errorf("get average price %w", errElementExceedsDataLength)
	}
	return (o.Open[element] + o.High[element] + o.Low[element] + o.Close[element]) / 4, nil
}

// GetTWAP returns the time weighted average price for the specified period.
// NOTE: This assumes the most recent price is at the tail end of the slice.
// Based off: https://blog.quantinsti.com/twap/
// Only returns one item as all other items are just the average price.
func (k *Item) GetTWAP() (float64, error) {
	if len(k.Candles) == 0 {
		return 0, fmt.Errorf("get twap %w", errNoData)
	}
	var cumAveragePrice float64
	for x := range k.Candles {
		cumAveragePrice += k.Candles[x].GetAveragePrice()
	}
	return cumAveragePrice / float64(len(k.Candles)), nil
}

// GetTWAP returns the time weighted average price for the specified period.
func (o *OHLC) GetTWAP() (float64, error) {
	if o == nil {
		return 0, fmt.Errorf("get twap %w", errNilOHLC)
	}
	if len(o.Open) == 0 || len(o.High) == 0 || len(o.Low) == 0 || len(o.Close) == 0 {
		return 0, fmt.Errorf("get twap %w", errNoData)
	}
	if len(o.Open) != len(o.High) || len(o.Open) != len(o.Low) || len(o.Open) != len(o.Close) {
		return 0, fmt.Errorf("get twap %w", errDataLengthMismatch)
	}

	var cumAveragePrice float64
	for x := range o.Close {
		avgPrice, err := o.GetAveragePrice(x)
		if err != nil {
			return 0, fmt.Errorf("get twap %w", err)
		}
		cumAveragePrice += avgPrice
	}
	return cumAveragePrice / float64(len(o.Close)), nil
}

// GetTypicalPrice returns the typical average price from the high, low and
// close values.
func (c *Candle) GetTypicalPrice() float64 {
	return (c.High + c.Low + c.Close) / 3
}

// GetTypicalPrice returns the typical average price from the high, low and
// close values.
func (o *OHLC) GetTypicalPrice(element int) (float64, error) {
	if o == nil {
		return 0, fmt.Errorf("get typical price %w", errNilOHLC)
	}
	if element < 0 {
		return 0, fmt.Errorf("get typical price %w", errInvalidElement)
	}
	check := element + 1
	if check > len(o.High) || check > len(o.Low) || check > len(o.Close) {
		return 0, fmt.Errorf("get typical price %w", errElementExceedsDataLength)
	}
	return (o.High[element] + o.Low[element] + o.Close[element]) / 3, nil
}

// GetVWAPs returns the Volume Weighted Averages prices which are the cumulative
// average price with respect to the volume.
// NOTE: This assumes candles are sorted by time
// Based off: https://blog.quantinsti.com/vwap-strategy/
func (k *Item) GetVWAPs() ([]float64, error) {
	if len(k.Candles) == 0 {
		return nil, fmt.Errorf("get vwap %w", errNoData)
	}
	store := make([]float64, len(k.Candles))
	var cumTotal, cumVolume float64
	for x := range k.Candles {
		cumTotal += k.Candles[x].GetTypicalPrice() * k.Candles[x].Volume
		cumVolume += k.Candles[x].Volume
		store[x] = cumTotal / cumVolume
	}
	return store, nil
}

// GetVWAPs returns the Volume Weighted Averages prices which are the cumulative
// average price with respect to the volume.
func (o *OHLC) GetVWAPs() ([]float64, error) {
	if o == nil {
		return nil, fmt.Errorf("get vwap %w", errNilOHLC)
	}
	if len(o.Open) == 0 || len(o.High) == 0 || len(o.Low) == 0 || len(o.Close) == 0 {
		return nil, fmt.Errorf("get vwap %w", errNoData)
	}
	if len(o.Open) != len(o.High) || len(o.Open) != len(o.Low) || len(o.Open) != len(o.Close) {
		return nil, fmt.Errorf("get vwap %w", errDataLengthMismatch)
	}

	store := make([]float64, len(o.High))
	var cumTotal, cumVolume float64
	for x := range o.High {
		typPrice, err := o.GetTypicalPrice(x)
		if err != nil {
			return nil, fmt.Errorf("get vwap %w", err)
		}
		cumTotal += typPrice * o.Volume[x]
		cumVolume += o.Volume[x]
		store[x] = cumTotal / cumVolume
	}
	return store, nil
}
