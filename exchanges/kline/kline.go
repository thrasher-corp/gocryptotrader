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

// FillMissingDataWithEmptyEntries amends a kline item to have candle entries
// for every interval between its start and end dates derived from ranges
func (k *Item) FillMissingDataWithEmptyEntries(i *IntervalRangeHolder) {
	var anyChanges bool
	for x := range i.Ranges {
		for y := range i.Ranges[x].Intervals {
			if !i.Ranges[x].Intervals[y].HasData {
				for z := range k.Candles {
					if i.Ranges[x].Intervals[y].Start.Equal(k.Candles[z].Time) {
						break
					}
				}
				anyChanges = true
				k.Candles = append(k.Candles, Candle{
					Time: i.Ranges[x].Intervals[y].Start.Time,
				})
			}
		}
	}
	if anyChanges {
		k.SortCandlesByTimestamp(false)
	}
}

// RemoveDuplicates removes any duplicate candles
func (k *Item) RemoveDuplicates() {
	var newCandles []Candle
	for x := range k.Candles {
		if x == 0 {
			newCandles = append(newCandles, k.Candles[x])
			continue
		}
		if !k.Candles[x].Time.Equal(k.Candles[x-1].Time) {
			// don't add duplicate
			newCandles = append(newCandles, k.Candles[x])
		}
	}

	k.Candles = newCandles
}

// RemoveOutsideRange removes any candles outside the start and end date
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
func TotalCandlesPerInterval(start, end time.Time, interval Interval) (out float64) {
	switch interval {
	case FifteenSecond:
		return end.Sub(start).Seconds() / 15
	case OneMin:
		return end.Sub(start).Minutes()
	case ThreeMin:
		return end.Sub(start).Minutes() / 3
	case FiveMin:
		return end.Sub(start).Minutes() / 5
	case TenMin:
		return end.Sub(start).Minutes() / 10
	case FifteenMin:
		return end.Sub(start).Minutes() / 15
	case ThirtyMin:
		return end.Sub(start).Minutes() / 30
	case OneHour:
		return end.Sub(start).Hours()
	case TwoHour:
		return end.Sub(start).Hours() / 2
	case FourHour:
		return end.Sub(start).Hours() / 4
	case SixHour:
		return end.Sub(start).Hours() / 6
	case EightHour:
		return end.Sub(start).Hours() / 8
	case TwelveHour:
		return end.Sub(start).Hours() / 12
	case OneDay:
		return end.Sub(start).Hours() / 24
	case ThreeDay:
		return end.Sub(start).Hours() / 72
	case FifteenDay:
		return end.Sub(start).Hours() / (24 * 15)
	case OneWeek:
		return end.Sub(start).Hours() / (24 * 7)
	case TwoWeek:
		return end.Sub(start).Hours() / (24 * 14)
	case OneMonth:
		return end.Sub(start).Hours() / (24 * 30)
	case OneYear:
		return end.Sub(start).Hours() / 8760
	}
	return -1
}

// IntervalsPerYear helps determine the number of intervals in a year
// used in CAGR calculation to know the amount of time of an interval in a year
func (i *Interval) IntervalsPerYear() float64 {
	return float64(OneYear.Duration().Nanoseconds()) / float64(i.Duration().Nanoseconds())
}

// CalculateCandleDateRanges will calculate the expected candle data in intervals in a date range
// If an API is limited in the amount of candles it can make in a request, it will automatically separate
// ranges into the limit
func CalculateCandleDateRanges(start, end time.Time, interval Interval, limit uint32) (*IntervalRangeHolder, error) {
	if err := common.StartEndTimeCheck(start, end); err != nil && !errors.Is(err, common.ErrStartAfterTimeNow) {
		return nil, err
	}
	if interval <= 0 {
		return nil, ErrUnsetInterval
	}

	start = start.Round(interval.Duration())
	end = end.Round(interval.Duration())
	resp := &IntervalRangeHolder{
		Start: CreateIntervalTime(start),
		End:   CreateIntervalTime(end),
	}
	var intervalsInWholePeriod []IntervalData
	for i := start; !i.After(end) && !i.Equal(end); i = i.Add(interval.Duration()) {
		intervalsInWholePeriod = append(intervalsInWholePeriod, IntervalData{
			Start: CreateIntervalTime(i.Round(interval.Duration())),
			End:   CreateIntervalTime(i.Round(interval.Duration()).Add(interval.Duration())),
		})
	}
	if len(intervalsInWholePeriod) < int(limit) || limit == 0 {
		resp.Ranges = []IntervalRange{{
			Start:     CreateIntervalTime(start),
			End:       CreateIntervalTime(end),
			Intervals: intervalsInWholePeriod,
		}}
		return resp, nil
	}

	var intervals []IntervalData
	splitIntervalsByLimit := make([][]IntervalData, 0, len(intervalsInWholePeriod)/int(limit)+1)
	for len(intervalsInWholePeriod) >= int(limit) {
		intervals, intervalsInWholePeriod = intervalsInWholePeriod[:limit], intervalsInWholePeriod[limit:]
		splitIntervalsByLimit = append(splitIntervalsByLimit, intervals)
	}
	if len(intervalsInWholePeriod) > 0 {
		splitIntervalsByLimit = append(splitIntervalsByLimit, intervalsInWholePeriod)
	}

	for x := range splitIntervalsByLimit {
		resp.Ranges = append(resp.Ranges, IntervalRange{
			Start:     splitIntervalsByLimit[x][0].Start,
			End:       splitIntervalsByLimit[x][len(splitIntervalsByLimit[x])-1].End,
			Intervals: splitIntervalsByLimit[x],
		})
	}

	return resp, nil
}

// HasDataAtDate determines whether a there is any data at a set
// date inside the existing limits
func (h *IntervalRangeHolder) HasDataAtDate(t time.Time) bool {
	tu := t.Unix()
	if tu < h.Start.Ticks || tu > h.End.Ticks {
		return false
	}
	for i := range h.Ranges {
		if tu >= h.Ranges[i].Start.Ticks && tu <= h.Ranges[i].End.Ticks {
			for j := range h.Ranges[i].Intervals {
				if tu >= h.Ranges[i].Intervals[j].Start.Ticks && tu < h.Ranges[i].Intervals[j].End.Ticks {
					return h.Ranges[i].Intervals[j].HasData
				}
				if j == len(h.Ranges[i].Intervals)-1 {
					if tu == h.Ranges[i].Start.Ticks {
						return h.Ranges[i].Intervals[j].HasData
					}
				}
			}
		}
	}

	return false
}

// SetHasDataFromCandles will calculate whether there is data in each candle
// allowing any missing data from an API request to be highlighted
func (h *IntervalRangeHolder) SetHasDataFromCandles(c []Candle) {
	for x := range h.Ranges {
	intervals:
		for y := range h.Ranges[x].Intervals {
			for z := range c {
				cu := c[z].Time.Unix()
				if cu >= h.Ranges[x].Intervals[y].Start.Ticks && cu < h.Ranges[x].Intervals[y].End.Ticks {
					h.Ranges[x].Intervals[y].HasData = true
					continue intervals
				}
			}
			h.Ranges[x].Intervals[y].HasData = false
		}
	}
}

// DataSummary returns a summary of a data range to highlight where data is missing
func (h *IntervalRangeHolder) DataSummary(includeHasData bool) []string {
	var (
		rangeStart, rangeEnd, prevStart, prevEnd time.Time
		rangeHasData                             bool
		rangeTexts                               []string
	)
	rangeStart = h.Start.Time
	for i := range h.Ranges {
		for j := range h.Ranges[i].Intervals {
			if h.Ranges[i].Intervals[j].HasData {
				if !rangeHasData && !rangeEnd.IsZero() {
					rangeTexts = append(rangeTexts, h.createDateSummaryRange(rangeStart, rangeEnd, rangeHasData))
					prevStart = rangeStart
					prevEnd = rangeEnd
					rangeStart = h.Ranges[i].Intervals[j].Start.Time
				}
				rangeHasData = true
			} else {
				if rangeHasData && !rangeEnd.IsZero() {
					if includeHasData {
						rangeTexts = append(rangeTexts, h.createDateSummaryRange(rangeStart, rangeEnd, rangeHasData))
					}
					prevStart = rangeStart
					prevEnd = rangeEnd
					rangeStart = h.Ranges[i].Intervals[j].Start.Time
				}
				rangeHasData = false
			}
			rangeEnd = h.Ranges[i].Intervals[j].End.Time
		}
	}
	if !rangeStart.Equal(prevStart) || !rangeEnd.Equal(prevEnd) {
		if (rangeHasData && includeHasData) || !rangeHasData {
			rangeTexts = append(rangeTexts, h.createDateSummaryRange(rangeStart, rangeEnd, rangeHasData))
		}
	}
	return rangeTexts
}

func (h *IntervalRangeHolder) createDateSummaryRange(start, end time.Time, hasData bool) string {
	dataString := "missing"
	if hasData {
		dataString = "has"
	}

	return fmt.Sprintf("%s data between %s and %s",
		dataString,
		start.Format(common.SimpleTimeFormat),
		end.Format(common.SimpleTimeFormat))
}

// CreateIntervalTime is a simple helper function to set the time twice
func CreateIntervalTime(tt time.Time) IntervalTime {
	return IntervalTime{
		Time:  tt,
		Ticks: tt.Unix(),
	}
}

// Equal allows for easier unix comparison
func (i *IntervalTime) Equal(tt time.Time) bool {
	return tt.Unix() == i.Ticks
}
