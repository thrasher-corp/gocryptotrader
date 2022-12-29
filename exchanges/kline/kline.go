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
func CreateKline(trades []order.TradeHistory, interval Interval, pair currency.Pair, a asset.Item, exchName string) (*Item, error) {
	if interval < FifteenSecond {
		return nil, fmt.Errorf("%w: [%s]", ErrInvalidInterval, interval)
	}

	if err := validateData(trades); err != nil {
		return nil, err
	}

	// Assuming the first trade is *actually* the first trade executed via
	// matching engine within this candle. e.g. For a block of trades that takes
	// place from 12:30 to 17:30 UTC, the data will be converted into hourly
	// candles that are aligned with UTC. The resulting candles will have an
	// open time of 12:00 and a close time of 17:59.9999 (17:00 open time). This
	// means that the first and last candles in this 6-hour window will have
	// half an hour of trading activity missing.
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
	// really necessary for NS prec because we are only fitting in >=minute
	// candles but for future custom candles we can open up a <=100ms heartbeat
	// if needed.
	candleWindowNs := interval.Duration().Nanoseconds()

	var offset int
	for x := range candles {
		if candles[x].Time.IsZero() {
			candles[x].Time = timeSeriesStart
			timeSeriesStart = timeSeriesStart.Add(interval.Duration())
		}
		candleStartNs := candles[x].Time.UnixNano()
		for y := offset; y < len(trades); y++ {
			if (trades[y].Timestamp.UnixNano() - candleStartNs) >= candleWindowNs {
				// Push forward offset
				offset = y
				break
			}

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
		return errInsufficientTradeData
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

// addPadding inserts padding time aligned when exchanges do not supply all data
// when there is no activity in a certain time interval.
// Start defines the request start and due to potential no activity from this
// point onwards this needs to be specified. ExclusiveEnd defines the end date
// which does not include a candle so everything from start can essentially be
// added with blank spaces.
func (k *Item) addPadding(start, exclusiveEnd time.Time) error {
	if k == nil {
		return errNilKline
	}

	if k.Interval <= 0 {
		return ErrInvalidInterval
	}

	window := exclusiveEnd.Sub(start)
	if window <= 0 {
		return errCannotEstablishTimeWindow
	}

	segments := int(window / k.Interval.Duration())
	if segments == len(k.Candles) {
		return nil
	}

	padded := make([]Candle, segments)
	var target int
	for x := range padded {
		if target >= len(k.Candles) || !k.Candles[target].Time.Equal(start) {
			padded[x].Time = start
		} else {
			padded[x] = k.Candles[target]
			target++
		}
		start = start.Add(k.Interval.Duration())
	}
	k.Candles = padded
	return nil
}

// RemoveDuplicates removes any duplicate candles. NOTE: Filter-in-place is used
// in this function for optimization and to keep the slice reference pointer the
// same, if changed ExtendedRequest ConvertCandles functionality will break.
func (k *Item) RemoveDuplicates() {
	lookup := make(map[int64]bool)
	target := 0
	for _, keep := range k.Candles {
		if key := keep.Time.Unix(); !lookup[key] {
			lookup[key] = true
			k.Candles[target] = keep
			target++
		}
	}
	k.Candles = k.Candles[:target]
}

// RemoveOutsideRange removes any candles outside the start and end date.
// NOTE: Filter-in-place is used in this function for optimization and to keep
// the slice reference pointer the same, if changed ExtendedRequest
// ConvertCandles functionality will break.
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
func (i Interval) IntervalsPerYear() float64 {
	if i == 0 {
		return 0
	}
	return oneYearDurationInNano / float64(i)
}

// ConvertToNewInterval allows the scaling of candles to larger candles
// e.g. Convert OneDay candles to ThreeDay candles, if there are adequate
// candles. Incomplete candles are NOT converted e.g. 4 OneDay candles will
// convert to one ThreeDay candle, skipping the fourth.
func (k *Item) ConvertToNewInterval(newInterval Interval) (*Item, error) {
	if k == nil {
		return nil, errNilKline
	}
	if k.Interval <= 0 {
		return nil, fmt.Errorf("%w for old candle", ErrInvalidInterval)
	}
	if newInterval <= 0 {
		return nil, fmt.Errorf("%w for new candle", ErrInvalidInterval)
	}
	if newInterval <= k.Interval {
		return nil, fmt.Errorf("%w %s is less than or equal to %s",
			ErrCanOnlyUpscaleCandles,
			newInterval,
			k.Interval)
	}
	if newInterval%k.Interval != 0 {
		return nil, fmt.Errorf("%s %w %s",
			k.Interval,
			ErrWholeNumberScaling,
			newInterval)
	}

	start := k.Candles[0].Time
	end := k.Candles[len(k.Candles)-1].Time.Add(k.Interval.Duration())
	window := end.Sub(start)
	if expected := int(window / k.Interval.Duration()); expected != len(k.Candles) {
		return nil, fmt.Errorf("%w expected candles %d but have only %d when converting from %s to %s interval",
			errCandleDataNotPadded,
			expected,
			len(k.Candles),
			k.Interval,
			newInterval)
	}

	oldIntervalsPerNewCandle := int(newInterval / k.Interval)
	candles := make([]Candle, len(k.Candles)/oldIntervalsPerNewCandle)
	if len(candles) == 0 {
		return nil, fmt.Errorf("%w to %v no candle data", ErrInsufficientCandleData, newInterval)
	}
	var target int
	for x := range k.Candles {
		if candles[target].Time.IsZero() {
			candles[target].Time = k.Candles[x].Time
		}

		if candles[target].Open == 0 {
			candles[target].Open = k.Candles[x].Open
		}

		if k.Candles[x].High > candles[target].High {
			candles[target].High = k.Candles[x].High
		}

		if candles[target].Low == 0 || k.Candles[x].Low < candles[target].Low {
			candles[target].Low = k.Candles[x].Low
		}

		candles[target].Volume += k.Candles[x].Volume

		if (x+1)%oldIntervalsPerNewCandle == 0 {
			candles[target].Close = k.Candles[x].Close
			target++
			// Note: Below checks the length of the proceeding slice so we can
			// break instantly if we cannot make an entire candle. e.g. 60 min
			// candles in an hour candle and we have 59 minute candles left.
			// This entire procession is cleaved.
			if len(k.Candles[x:])-1 < oldIntervalsPerNewCandle {
				break
			}
		}
	}
	return &Item{
		Exchange: k.Exchange,
		Pair:     k.Pair,
		Asset:    k.Asset,
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
		return nil, ErrInvalidInterval
	}

	start = start.Round(interval.Duration())
	end = end.Round(interval.Duration())
	window := end.Sub(start)
	count := int64(window) / int64(interval)
	requests := float64(count) / float64(limit)

	switch {
	case requests <= 1:
		requests = 1
	case limit == 0:
		requests, limit = 1, uint32(count)
	case requests-float64(int64(requests)) > 0:
		requests++
	}

	potentialRequests := make([]IntervalRange, int(requests))
	requestStart := start
	for x := range potentialRequests {
		potentialRequests[x].Start = CreateIntervalTime(requestStart)

		count -= int64(limit)
		if count < 0 {
			potentialRequests[x].Intervals = make([]IntervalData, count+int64(limit))
		} else {
			potentialRequests[x].Intervals = make([]IntervalData, limit)
		}

		for y := range potentialRequests[x].Intervals {
			potentialRequests[x].Intervals[y].Start = CreateIntervalTime(requestStart)
			requestStart = requestStart.Add(interval.Duration())
			potentialRequests[x].Intervals[y].End = CreateIntervalTime(requestStart)
		}
		potentialRequests[x].End = CreateIntervalTime(requestStart)
	}
	return &IntervalRangeHolder{
		Start:  CreateIntervalTime(start),
		End:    CreateIntervalTime(requestStart),
		Ranges: potentialRequests,
		Limit:  int(limit),
	}, nil
}

// HasDataAtDate determines whether a there is any data at a set
// date inside the existing limits
func (h *IntervalRangeHolder) HasDataAtDate(t time.Time) bool {
	tu := t.Unix()
	if tu < h.Start.Ticks || tu > h.End.Ticks {
		return false
	}
	for i := range h.Ranges {
		if tu < h.Ranges[i].Start.Ticks || tu >= h.Ranges[i].End.Ticks {
			continue
		}

		for j := range h.Ranges[i].Intervals {
			if tu >= h.Ranges[i].Intervals[j].Start.Ticks &&
				tu < h.Ranges[i].Intervals[j].End.Ticks {
				return h.Ranges[i].Intervals[j].HasData
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

// ExchangeSupported returns if the exchange directly supports the interval. In
// future this might be able to be deprecated because we can construct custom
// intervals from the supported list.
func (e *ExchangeIntervals) ExchangeSupported(in Interval) bool {
	return e.supported[in]
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
