package twap

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var errNoKlineData = errors.New("no kline data")

type Candles []kline.Candle

// Merge merges two candle datasets together
func (c *Candles) Merge(incoming []kline.Candle) error {
	if len(incoming) == 0 {
		return errNoKlineData
	}

	// var offset int
	// // Only using the first element
	// for x := range incoming {
	// 	// Go backwards to find time match as this can be quite a large set and
	// 	// incoming data should be new data.
	// 	for y := len(*c); y > -1; y-- {
	// 		if incoming[x].Time.Equal((*c)[y].Time) {
	// 			offset = y
	// 		}
	// 	}
	// }

	return nil
}

// GetLookBack returns the look back time from the specified interval and the
// requested period length.
func GetLookBack(in kline.Interval, length int) (time.Time, error) {
	tn := time.Now().UTC()
	for ; length > -1; length-- {
		tn = tn.Add(-in.Duration())
	}
	return tn, nil
}
