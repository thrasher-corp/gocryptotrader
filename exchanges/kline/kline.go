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

var (
	// ErrInvalidInterval defines when an interval is invalid e.g. interval <= 0
	ErrInvalidInterval = errors.New("invalid interval")
	// ErrCannotConstructInterval defines an error when an interval cannot be
	// constructed from a list of support intervals.
	ErrCannotConstructInterval = errors.New("cannot construct required interval from supported intervals")
	// ErrInsufficientCandleData defines an error when you have a candle that
	// requires multiple candles to generate.
	ErrInsufficientCandleData = errors.New("insufficient candle data to generate new candle")

	oneYearDurationInNano = float64(OneYear.Duration().Nanoseconds())
)

// CreateKline creates candles out of trade history data for a set time interval
func CreateKline(trades []order.TradeHistory, interval Interval, pair currency.Pair, a asset.Item, exchName string) (*Item, error) {
	if interval.Duration() < time.Minute {
		return nil, fmt.Errorf("invalid time interval: [%s]", interval)
	}

	if err := validateData(trades); err != nil {
		return nil, err
	}

	// Assuming the first trade is *actually* the first trade executed via
	// matching engine within this candle.
	timeSeriesStart := trades[0].Timestamp.Truncate(interval.Duration())

	// Assuming the last trade is *actually* the last trade executed via
	// matching engine within this candle.
	timeSeriesEnd := trades[len(trades)-1].Timestamp.Truncate(interval.Duration()).Add(interval.Duration())

	// Full duration window or block for which all trades will occur.
	window := timeSeriesEnd.Sub(timeSeriesStart)

	count := int64(window) / int64(interval)

	// Opted to create blanks in memory so that if no trading occurs we don't
	// need to insert a blank candle later.
	candles := make([]Candle, count)

	// Opted for arithmetic operations for trade candle matching. It's not
	// really neccesary for NS prec because we are only fitting in >=minute
	// candles but for future custom candles we can open up a <=100ms heartbeat
	// if needed.
	candleWindowNs := interval.Duration().Nanoseconds()

	for x := range candles {
		if candles[x].Time.IsZero() {
			candles[x].Time = timeSeriesStart
			timeSeriesStart = timeSeriesStart.Add(interval.Duration())
		}
		candleStartNs := candles[x].Time.UnixNano()
		for y := range trades {
			if (trades[y].Timestamp.UnixNano() - candleStartNs) < candleWindowNs {
				if candles[x].Open == 0 {
					candles[x].Open = trades[y].Price
				}
				if candles[x].High < trades[y].Price {
					candles[x].High = trades[y].Price
				}

				if candles[x].Low == 0 || candles[x].Low > trades[y].Price {
					candles[x].Low = trades[y].Price
				}

				candles[x].Close = trades[y].Price
				candles[x].Volume += trades[y].Amount
				continue
			}
			// Cleave used trades for faster walk. TODO: Might need to copy a
			// full trade slice so that we don't purge the param reference.
			trades = trades[y:]
			break
		}
	}

	return &Item{
		Exchange: exchName,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
		Candles:  candles,
	}, nil
}

// validateData checks for zero values on data and sorts before turning
// converting into OHLC
func validateData(trades []order.TradeHistory) error {
	if len(trades) < 2 {
		return errors.New("insufficient data")
	}

	for i := range trades {
		if trades[i].Timestamp.IsZero() || trades[i].Timestamp.Unix() == 0 {
			return fmt.Errorf("timestamp not set for element %d", i)
		}

		if trades[i].Amount <= 0 {
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
// Filter in place must be used as it keeps the slice reference pointer the same.
// If changed BuilderExteneded ConvertCandles functionality will break.
func (k *Item) RemoveDuplicates() {
	lookup := make(map[int64]bool)
	target := 0
	for _, keep := range k.Candles {
		key := keep.Time.Unix()
		if lookup[key] {
			continue
		}
		lookup[key] = true
		k.Candles[target] = keep
		target++
	}
	k.Candles = k.Candles[:target]
}

// RemoveOutsideRange removes any candles outside the start and end date
// Filter in place must be used as it keeps the slice reference pointer the same.
// If changed BuilderExteneded ConvertCandles functionality will break.
func (k *Item) RemoveOutsideRange(start, end time.Time) {
	target := 0
	for _, keep := range k.Candles {
		if keep.Time.Equal(start) || (keep.Time.After(start) && keep.Time.Before(end)) {
			k.Candles[target] = keep
			target++
		}
	}
	k.Candles = k.Candles[:target]
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

// FormatDates converts all dates to UTC time
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
func TotalCandlesPerInterval(start, end time.Time, interval Interval) int64 {
	if interval <= 0 {
		return 0
	}
	window := end.Sub(start)
	return int64(window) / int64(interval)
}

// IntervalsPerYear helps determine the number of intervals in a year
// used in CAGR calculation to know the amount of time of an interval in a year
func (i *Interval) IntervalsPerYear() float64 {
	if *i == 0 {
		return 0
	}
	return oneYearDurationInNano / float64(i.Duration().Nanoseconds())
}

// ConvertToNewInterval allows the scaling of candles to larger candles
// e.g. Convert OneDay candles to ThreeDay candles, if there are adequate
// candles. Incomplete candles are NOT converted e.g. an 4 OneDay candles will
// convert to one ThreeDay candle, skipping the fourth.
func ConvertToNewInterval(old *Item, newInterval Interval) (*Item, error) {
	if old == nil {
		return nil, errNilKline
	}
	if newInterval <= 0 {
		return nil, ErrUnsetInterval
	}
	if newInterval <= old.Interval {
		return nil, ErrCanOnlyDownscaleCandles
	}
	if newInterval%old.Interval != 0 {
		return nil, ErrWholeNumberScaling
	}

	oldIntervalsPerNewCandle := int(newInterval / old.Interval)
	candles := make([]Candle, len(old.Candles)/oldIntervalsPerNewCandle)
	if len(candles) == 0 {
		return nil, ErrInsufficientCandleData
	}
	var target int
	for x := range old.Candles {
		if candles[target].Time.IsZero() {
			candles[target].Time = old.Candles[x].Time
			candles[target].Open = old.Candles[x].Open
		}

		if old.Candles[x].High > candles[target].High {
			candles[target].High = old.Candles[x].High
		}

		if candles[target].Low == 0 || candles[target].Low != 0 && old.Candles[x].Low < candles[target].Low {
			candles[target].Low = old.Candles[x].Low
		}

		candles[target].Volume += old.Candles[x].Volume

		if (x+1)%oldIntervalsPerNewCandle == 0 {
			candles[target].Close = old.Candles[x].Close
			target++
			// Note: Below checks the length of the proceeding slice so we can
			// break instantly if we cannot make an entire candle. e.g. 60 min
			// candles in an hour candle and we have 59 minute candles left.
			// This entire procession is cleaved.
			if len(old.Candles[x:])-1 < oldIntervalsPerNewCandle {
				break
			}
		}
	}
	return &Item{
		Exchange: old.Exchange,
		Pair:     old.Pair,
		Asset:    old.Asset,
		Interval: newInterval,
		Candles:  candles,
	}, nil
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
	for i := start; i.Before(end) && !i.Equal(end); i = i.Add(interval.Duration()) {
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

// GetClosePriceAtTime returns the close price of a candle
// at a given time
func (k *Item) GetClosePriceAtTime(t time.Time) (float64, error) {
	for i := range k.Candles {
		if k.Candles[i].Time.Equal(t) {
			return k.Candles[i].Close, nil
		}
	}
	return -1, fmt.Errorf("%w at %v", ErrNotFoundAtTime, t)
}

// SetHasDataFromCandles will calculate whether there is data in each candle
// allowing any missing data from an API request to be highlighted
func (h *IntervalRangeHolder) SetHasDataFromCandles(incoming []Candle) {
	bucket := make([]Candle, len(incoming))
	copy(bucket, incoming)
	for x := range h.Ranges {
	intervals:
		for y := range h.Ranges[x].Intervals {
			for z := range bucket {
				cu := bucket[z].Time.Unix()
				if cu >= h.Ranges[x].Intervals[y].Start.Ticks &&
					cu < h.Ranges[x].Intervals[y].End.Ticks {
					h.Ranges[x].Intervals[y].HasData = true
					bucket = bucket[z+1:]
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
	return IntervalTime{Time: tt, Ticks: tt.Unix()}
}

// Equal allows for easier unix comparison
func (i *IntervalTime) Equal(tt time.Time) bool {
	return tt.Unix() == i.Ticks
}

// DeployExchangeIntervals aligns and stores supported intervals for an exchange
// for future matching.
func DeployExchangeIntervals(enabled ...Interval) ExchangeIntervals {
	sort.Slice(enabled, func(i, j int) bool { return enabled[i] < enabled[j] })

	supported := make(map[Interval]bool)
	for x := range enabled {
		supported[enabled[x]] = true
	}
	return ExchangeIntervals{supported: supported, aligned: enabled}
}

// Supports returns if the exchange directly supports the interval. In future
// this might be able to be deprecated because we can construct custom intervals
// from the supported list.
func (e *ExchangeIntervals) Supports(required Interval) bool {
	return e.supported[required]
}

// Construct fetches supported interval that can construct the required interval
// e.g. 1 hour interval candles can be made from 2 * 30 minute interval candles.
func (e *ExchangeIntervals) Construct(required Interval) (Interval, error) {
	if required <= 0 {
		return 0, ErrInvalidInterval
	}

	if e.supported[required] {
		// Directly supported by exchange can return.
		return required, nil
	}

	for x := len(e.aligned) - 1; x > -1; x-- {
		if e.aligned[x] < required && required%e.aligned[x] == 0 {
			// Indirectly supported by exchange. Can generate required candle
			// from this lower time frame supported candle.
			return e.aligned[x], nil
		}
	}
	return 0, ErrCannotConstructInterval
}
