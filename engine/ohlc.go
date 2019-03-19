package engine

import (
	"errors"
	"fmt"
	"sort"
	"time"

	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Candles is the main return type for this package with attached methods for
// manipulating the sliced data
type Candles []Candle

// Candle contains collated data for differing time periods
type Candle struct {
	PercentageChange float64
	TimePeriod       time.Duration
	High             float64
	Low              float64
	Close            float64
	CloseTime        time.Time
	Open             float64
	OpenTime         time.Time
	Volume           float64
	Validation       []exchange.PlatformTrade
}

// HeartBeat denotes open and close times on a chart respective to time period
type HeartBeat struct {
	Open  time.Time
	Close time.Time
}

// CreateOHLC creates candles out of trade history data for a set time period
func CreateOHLC(h PlatformHistory, timePeriod time.Duration) (Candles, error) {
	// Determines if we have zero value entries
	err := h.ValidatData()
	if err != nil {
		return nil, err
	}

	err = h.Sort()
	if err != nil {
		return nil, err
	}

	var candles []Candle
	timeIntervalStart := h[0].Timestamp.Truncate(timePeriod)
	timeIntervalEnd := h[len(h)-1].Timestamp

	// Adds time interval buffer zones
	var timeIntervalCache [][]*exchange.PlatformTrade
	var OpenClose []HeartBeat

	for t := timeIntervalStart; t.Before(timeIntervalEnd); t = t.Add(timePeriod) {
		timeBufferEnd := t.Add(timePeriod)
		insertionCount := 0

		var zonedTradeHistory []*exchange.PlatformTrade
		for i := 0; i < len(h); i++ {
			if (h[i].Timestamp.After(t) || h[i].Timestamp.Equal(t)) &&
				(h[i].Timestamp.Before(timeBufferEnd) ||
					h[i].Timestamp.Equal(timeBufferEnd)) {
				zonedTradeHistory = append(zonedTradeHistory, h[i])
				insertionCount++
				continue
			}
			h = h[i:]
			break
		}

		// Insert dummy in time period when there is no price action
		if insertionCount == 0 {
			OpenClose = append(OpenClose, HeartBeat{Open: t, Close: timeBufferEnd})
			timeIntervalCache = append(timeIntervalCache, []*exchange.PlatformTrade{})
			continue
		}
		OpenClose = append(OpenClose, HeartBeat{Open: t, Close: timeBufferEnd})
		timeIntervalCache = append(timeIntervalCache, zonedTradeHistory)
	}

	var closePriceOfLast float64
	for x := range timeIntervalCache {
		if len(timeIntervalCache[x]) == 0 {
			candles = append(candles, Candle{
				OpenTime:   OpenClose[x].Open,
				CloseTime:  OpenClose[x].Close,
				High:       closePriceOfLast,
				Low:        closePriceOfLast,
				Close:      closePriceOfLast,
				Open:       closePriceOfLast,
				TimePeriod: timePeriod})
			continue
		}

		var newCandle Candle
		for y := range timeIntervalCache[x] {
			if y == 0 {
				newCandle.Open = timeIntervalCache[x][y].Price
				newCandle.OpenTime = OpenClose[x].Open
			}
			if y == len(timeIntervalCache[x])-1 {
				newCandle.Close = timeIntervalCache[x][y].Price
				closePriceOfLast = timeIntervalCache[x][y].Price
				newCandle.CloseTime = OpenClose[x].Close
			}
			if newCandle.High < timeIntervalCache[x][y].Price {
				newCandle.High = timeIntervalCache[x][y].Price
			}
			if newCandle.Low > timeIntervalCache[x][y].Price || newCandle.Low == 0 {
				newCandle.Low = timeIntervalCache[x][y].Price
			}
			newCandle.Volume += timeIntervalCache[x][y].Amount
			newCandle.TimePeriod = timePeriod
		}
		newCandle.PercentageChange = (newCandle.Close - newCandle.Open) / newCandle.Open * 100
		candles = append(candles, newCandle)
	}
	return candles, nil
}

// PlatformHistory is used to attach methods for sorting trade history
// TODO: add to exchange package
type PlatformHistory []*exchange.PlatformTrade

// Forward request for length
func (t PlatformHistory) Len() int {
	return len(t)
}

// Define compare
func (t PlatformHistory) Less(i, j int) bool {
	return t[i].Timestamp.Before(t[j].Timestamp)
}

// Define swap over an array
func (t PlatformHistory) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// Sort sorts the trading history into ascending time
func (t PlatformHistory) Sort() error {
	if len(t) < 2 {
		return errors.New("insufficient data to sort")
	}
	sort.Sort(t)
	return nil
}

// ValidatData checks for zero values on data
func (t PlatformHistory) ValidatData() error {
	if len(t) == 0 {
		return errors.New("insufficient data to validate")
	}

	for i := range t {
		if t[i].Timestamp.IsZero() || t[i].Timestamp.Unix() == 0 {
			return fmt.Errorf("timestamp not set for element %d", i)
		}

		if t[i].Amount == 0 {
			return fmt.Errorf("amount not set for element %d", i)
		}

		if t[i].Price == 0 {
			return fmt.Errorf("price not set for element %d", i)
		}
	}
	return nil
}
