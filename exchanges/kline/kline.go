package kline

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// CreateKline creates candles out of trade history data for a set time interval
func CreateKline(trades []order.TradeHistory, interval Interval, p currency.Pair, a asset.Item, exchange string) (Item, error) {
	if interval.Duration() < time.Minute {
		return Item{}, fmt.Errorf("invalid time interval: [%s]", interval)
	}

	err := validateData(trades)
	if err != nil {
		return Item{}, err
	}

	timeIntervalStart := trades[0].Timestamp.Truncate(interval.Duration())
	timeIntervalEnd := trades[len(trades)-1].Timestamp

	// Adds time interval buffer zones
	var timeIntervalCache [][]order.TradeHistory
	var candleStart []time.Time

	for t := timeIntervalStart; t.Before(timeIntervalEnd); t = t.Add(interval.Duration()) {
		timeBufferEnd := t.Add(interval.Duration())
		insertionCount := 0

		var zonedTradeHistory []order.TradeHistory
		for i := 0; i < len(trades); i++ {
			if (trades[i].Timestamp.After(t) ||
				trades[i].Timestamp.Equal(t)) &&
				(trades[i].Timestamp.Before(timeBufferEnd) ||
					trades[i].Timestamp.Equal(timeBufferEnd)) {
				zonedTradeHistory = append(zonedTradeHistory, trades[i])
				insertionCount++
				continue
			}
			trades = trades[i:]
			break
		}

		candleStart = append(candleStart, t)

		// Insert dummy in time period when there is no price action
		if insertionCount == 0 {
			timeIntervalCache = append(timeIntervalCache, []order.TradeHistory{})
			continue
		}
		timeIntervalCache = append(timeIntervalCache, zonedTradeHistory)
	}

	if candleStart == nil {
		return Item{}, errors.New("candle start cannot be nil")
	}

	var candles = Item{
		Exchange: exchange,
		Pair:     p,
		Asset:    a,
		Interval: interval,
	}

	var closePriceOfLast float64
	for x := range timeIntervalCache {
		if len(timeIntervalCache[x]) == 0 {
			candles.Candles = append(candles.Candles, Candle{
				Time:  candleStart[x],
				High:  closePriceOfLast,
				Low:   closePriceOfLast,
				Close: closePriceOfLast,
				Open:  closePriceOfLast})
			continue
		}

		var newCandle = Candle{
			Open: timeIntervalCache[x][0].Price,
			Time: candleStart[x],
		}

		for y := range timeIntervalCache[x] {
			if y == len(timeIntervalCache[x])-1 {
				newCandle.Close = timeIntervalCache[x][y].Price
				closePriceOfLast = timeIntervalCache[x][y].Price
			}
			if newCandle.High < timeIntervalCache[x][y].Price {
				newCandle.High = timeIntervalCache[x][y].Price
			}
			if newCandle.Low > timeIntervalCache[x][y].Price || newCandle.Low == 0 {
				newCandle.Low = timeIntervalCache[x][y].Price
			}
			newCandle.Volume += timeIntervalCache[x][y].Amount
		}
		candles.Candles = append(candles.Candles, newCandle)
	}
	return candles, nil
}

// validateData checks for zero values on data and sorts before turning
// converting into OHLC
func validateData(trades []order.TradeHistory) error {
	if len(trades) < 2 {
		return errors.New("insufficient data")
	}

	for i := range trades {
		if trades[i].Timestamp.IsZero() ||
			trades[i].Timestamp.Unix() == 0 {
			return fmt.Errorf("timestamp not set for element %d", i)
		}

		if trades[i].Amount == 0 {
			return fmt.Errorf("amount not set for element %d", i)
		}

		if trades[i].Price == 0 {
			return fmt.Errorf("price not set for element %d", i)
		}
	}

	sort.Slice(trades, func(i, j int) bool {
		return trades[i].Timestamp.Before(trades[j].Timestamp)
	})
	return nil
}

// String returns numeric string
func (i Interval) String() string {
	return i.Duration().String()
}

// Word returns text version of Interval
func (i Interval) Word() string {
	return durationToWord(i)
}

// Duration returns interval casted as time.Duration for compatibility
func (i Interval) Duration() time.Duration {
	return time.Duration(i)
}

// Short returns short string version of interval
func (i Interval) Short() string {
	s := i.String()
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	if strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}

// durationToWord returns english version of interval
func durationToWord(in Interval) string {
	switch in {
	case FifteenSecond:
		return "fifteensecond"
	case OneMin:
		return "onemin"
	case ThreeMin:
		return "threemin"
	case FiveMin:
		return "fivemin"
	case TenMin:
		return "tenmin"
	case FifteenMin:
		return "fifteenmin"
	case ThirtyMin:
		return "thirtymin"
	case OneHour:
		return "onehour"
	case TwoHour:
		return "twohour"
	case FourHour:
		return "fourhour"
	case SixHour:
		return "sixhour"
	case EightHour:
		return "eighthour"
	case TwelveHour:
		return "twelvehour"
	case OneDay:
		return "oneday"
	case ThreeDay:
		return "threeday"
	case FifteenDay:
		return "fifteenday"
	case OneWeek:
		return "oneweek"
	case TwoWeek:
		return "twoweek"
	case OneMonth:
		return "onemonth"
	case OneYear:
		return "oneyear"
	default:
		return "notfound"
	}
}

// TotalCandlesPerInterval turns total candles per period for interval
func TotalCandlesPerInterval(start, end time.Time, interval Interval) (out uint32) {
	switch interval {
	case FifteenSecond:
		out = uint32(end.Sub(start).Seconds() / 15)
	case OneMin:
		out = uint32(end.Sub(start).Minutes())
	case ThreeMin:
		out = uint32(end.Sub(start).Minutes() / 3)
	case FiveMin:
		out = uint32(end.Sub(start).Minutes() / 5)
	case TenMin:
		out = uint32(end.Sub(start).Minutes() / 10)
	case FifteenMin:
		out = uint32(end.Sub(start).Minutes() / 15)
	case ThirtyMin:
		out = uint32(end.Sub(start).Minutes() / 30)
	case OneHour:
		out = uint32(end.Sub(start).Hours())
	case TwoHour:
		out = uint32(end.Sub(start).Hours() / 2)
	case FourHour:
		out = uint32(end.Sub(start).Hours() / 4)
	case SixHour:
		out = uint32(end.Sub(start).Hours() / 6)
	case EightHour:
		out = uint32(end.Sub(start).Hours() / 8)
	case TwelveHour:
		out = uint32(end.Sub(start).Hours() / 12)
	case OneDay:
		out = uint32(end.Sub(start).Hours() / 24)
	case ThreeDay:
		out = uint32(end.Sub(start).Hours() / 72)
	case FifteenDay:
		out = uint32(end.Sub(start).Hours() / (24 * 15))
	case OneWeek:
		out = uint32(end.Sub(start).Hours()) / (24 * 7)
	case TwoWeek:
		out = uint32(end.Sub(start).Hours() / (24 * 14))
	case OneMonth:
		out = uint32(end.Sub(start).Hours() / (24 * 30))
	case OneYear:
		out = uint32(end.Sub(start).Hours() / 8760)
	}
	return out
}

// CalcDateRanges returns slice of start/end times based on start & end date
func CalcDateRanges(start, end time.Time, interval Interval, limit uint32) (out []DateRange) {
	total := TotalCandlesPerInterval(start, end, interval)
	if total < limit {
		return []DateRange{{
			Start: start,
			End:   end,
		},
		}
	}
	var allDateIntervals []time.Time
	var y uint32
	var lastNum int
	for d := start; !d.After(end); d = d.Add(interval.Duration()) {
		allDateIntervals = append(allDateIntervals, d)
	}
	for x := range allDateIntervals {
		if y == limit {
			out = append(out, DateRange{
				allDateIntervals[x-int(limit)],
				allDateIntervals[x],
			})
			y = 0
			lastNum = x
		}
		y++
	}
	if allDateIntervals != nil && lastNum+1 < len(allDateIntervals) {
		out = append(out, DateRange{
			Start: allDateIntervals[lastNum+1],
			End:   allDateIntervals[len(allDateIntervals)-1],
		})
	}
	return out
}

// SortCandlesByTimestamp sorts candles by timestamp
func (k *Item) SortCandlesByTimestamp(desc bool) {
	sort.Slice(k.Candles, func(i, j int) bool {
		if desc {
			return k.Candles[i].Time.After(k.Candles[j].Time)
		}
		return k.Candles[i].Time.Before(k.Candles[j].Time)
	})
}

// FormatDates converts all date to UTC time
func (k *Item) FormatDates() {
	for x := range k.Candles {
		k.Candles[x].Time = k.Candles[x].Time.UTC()
	}
}
