package kline

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
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

// FillMissingDataWithEmptyEntries ammends a kline item to have candle entries
// for every interval between its start and end dates derived from ranges
func (k *Item) FillMissingDataWithEmptyEntries(i IntervalRangeHolder) {
	var anyChanges bool
	for x := range i.Ranges {
		for y := range i.Ranges[x].Intervals {
			if !i.Ranges[x].Intervals[y].HasData {
				for z := range k.Candles {
					if k.Candles[z].Time.Equal(i.Ranges[x].Intervals[y].Start) {
						break
					}
				}
				anyChanges = true
				k.Candles = append(k.Candles, Candle{
					Time: i.Ranges[x].Intervals[y].Start,
				})
			}
		}
	}
	if anyChanges {
		k.SortCandlesByTimestamp(false)
	}
}

// CalculateCandleDateRanges will calculate the expected candle data in intervals in a date range
// If an API is limited in the amount of candles it can make in a request, it will automatically separate
// ranges into the limit
func CalculateCandleDateRanges(start, end time.Time, interval Interval, limit uint32) IntervalRangeHolder {
	start = start.Round(interval.Duration())
	end = end.Round(interval.Duration())
	resp := IntervalRangeHolder{
		Start: start,
		End:   end,
	}
	var intervalsInWholePeriod []IntervalData
	for i := start; !i.After(end); i = i.Add(interval.Duration()) {
		intervalsInWholePeriod = append(intervalsInWholePeriod, IntervalData{
			Start: i.Round(interval.Duration()),
			End:   i.Round(interval.Duration()).Add(interval.Duration()),
		})
	}
	for intervalsInWholePeriod[len(intervalsInWholePeriod)-1].Start.After(end) || intervalsInWholePeriod[len(intervalsInWholePeriod)-1].Start.Equal(end) {
		// remove any extra intervals which have been added due to the "after"
		intervalsInWholePeriod = intervalsInWholePeriod[:len(intervalsInWholePeriod)-1]
	}
	if len(intervalsInWholePeriod) < int(limit) || limit == 0 {
		resp.Ranges = []IntervalRange{{
			Start:     start,
			End:       end,
			Intervals: intervalsInWholePeriod,
		}}
		return resp
	}

	var intervals []IntervalData
	splitIntervalsByLimit := make([][]IntervalData, 0, len(intervalsInWholePeriod)/int(limit)+1)
	for len(intervalsInWholePeriod) >= int(limit) {
		intervals, intervalsInWholePeriod = intervalsInWholePeriod[:limit], intervalsInWholePeriod[limit:]
		splitIntervalsByLimit = append(splitIntervalsByLimit, intervals)
	}
	if len(intervalsInWholePeriod) > 0 {
		splitIntervalsByLimit = append(splitIntervalsByLimit, intervalsInWholePeriod[:len(intervalsInWholePeriod)])
	}

	for x := range splitIntervalsByLimit {
		resp.Ranges = append(resp.Ranges, IntervalRange{
			Start:     splitIntervalsByLimit[x][0].Start,
			End:       splitIntervalsByLimit[x][len(splitIntervalsByLimit[x])-1].End,
			Intervals: splitIntervalsByLimit[x],
		})
	}

	return resp
}

// RemoveDuplicates removes any duplicate candles
func (k *Item) RemoveDuplicates() {
	var newCandles []Candle
	for x := range k.Candles {
		if x == 0 {
			continue
		}
		if !k.Candles[x].Time.Equal(k.Candles[x-1].Time) {
			// don't add duplicate
			newCandles = append(newCandles, k.Candles[x])
		}
	}

	k.Candles = newCandles
}

func (k *Item) RemoveOutsideRange(start, end time.Time) {
	var newCandles []Candle
	for i := range k.Candles {
		if k.Candles[i].Time.Equal(start) ||
			(k.Candles[i].Time.After(start) && k.Candles[i].Time.Before(end)) {
			newCandles = append(newCandles, k.Candles[i])
		}
	}
	k.Candles = newCandles
}

// HasDataAtDate determines whether a there is any data at a set
// date inside the existing limits
func (h *IntervalRangeHolder) HasDataAtDate(t time.Time) bool {
	if t.Before(t) || t.After(h.End) {
		return false
	}
	for i := range h.Ranges {
		if t.Equal(h.Ranges[i].Start) ||
			(t.After(h.Ranges[i].Start) && t.Before(h.Ranges[i].End)) {
			for j := range h.Ranges[i].Intervals {
				if t.Equal(h.Ranges[i].Intervals[j].Start) ||
					(t.After(h.Ranges[i].Intervals[j].Start) && t.Before(h.Ranges[i].Intervals[j].End)) {
					return h.Ranges[i].Intervals[j].HasData
				}
			}
		}
	}

	return false
}

// Verify will calculate whether there is data in each candle
// allowing any missing data from an API request to be highlighted
func (h *IntervalRangeHolder) Verify(c []Candle) error {
	for x := range h.Ranges {
		for y := range h.Ranges[x].Intervals {
			for z := range c {
				if c[z].Time.Equal(h.Ranges[x].Intervals[y].Start) ||
					(c[z].Time.After(h.Ranges[x].Intervals[y].Start) && c[z].Time.Before(h.Ranges[x].Intervals[y].End)) {
					h.Ranges[x].Intervals[y].HasData = true
				}
			}
		}
	}

	var errs common.Errors
	for x := range h.Ranges {
		for y := range h.Ranges[x].Intervals {
			if !h.Ranges[x].Intervals[y].HasData {
				errs = append(errs, fmt.Errorf("missing candles data between %v (%v) & %v (%v)", h.Ranges[x].Intervals[y].Start, h.Ranges[x].Intervals[y].Start.Unix(), h.Ranges[x].Intervals[y].End, h.Ranges[x].Intervals[y].End.Unix()))
			}
		}
	}
	if len(errs) > 0 {
		return errs
	}

	return nil
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
