package kline

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
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
	if i == Raw {
		return "raw"
	}
	s := i.String()
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	if strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}

// UnmarshalJSON implements the json.Unmarshaler interface for Intervals
// It does not validate the duration is aligned, only that it is a parsable duration
func (i *Interval) UnmarshalJSON(text []byte) error {
	text = bytes.Trim(text, `"`)
	if string(text) == "raw" {
		*i = Raw
		return nil
	}
	if len(bytes.TrimLeft(text, `0123456789`)) > 0 { // contains non-numerics, ParseDuration can handle errors
		d, err := time.ParseDuration(string(text))
		if err != nil {
			return err
		}
		*i = Interval(d)
	} else {
		n, err := strconv.ParseInt(string(text), 10, 64)
		if err != nil {
			return err
		}
		*i = Interval(n)
	}
	return nil
}

// MarshalText implements the TextMarshaler interface for Intervals
func (i Interval) MarshalText() ([]byte, error) {
	return []byte(i.Short()), nil
}

// addPadding inserts padding time aligned when exchanges do not supply all data
// when there is no activity in a certain time interval.
// Start defines the request start and due to potential no activity from this
// point onwards this needs to be specified. ExclusiveEnd defines the end date
// which does not include a candle so everything from start can essentially be
// added with blank spaces.
func (k *Item) addPadding(start, exclusiveEnd time.Time, purgeOnPartial bool) error {
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

	padded := make([]Candle, int(window/k.Interval.Duration()))
	var target int
	for x := range padded {
		switch {
		case target >= len(k.Candles):
			padded[x].Time = start
		case !k.Candles[target].Time.Equal(start):
			if k.Candles[target].Time.Before(start) {
				return fmt.Errorf("%w %q should be %q at %q interval",
					errCandleOpenTimeIsNotUTCAligned,
					k.Candles[target].Time,
					start.Add(k.Interval.Duration()),
					k.Interval)
			}
			padded[x].Time = start
		default:
			padded[x] = k.Candles[target]
			target++
		}
		start = start.Add(k.Interval.Duration())
	}

	// NOTE: This checks if the end time exceeds time.Now() and we are capturing
	// a partially created candle. This will only delete an element if it is
	// empty.
	if purgeOnPartial {
		lastElement := padded[len(padded)-1]
		if lastElement.Volume == 0 &&
			lastElement.Open == 0 &&
			lastElement.High == 0 &&
			lastElement.Low == 0 &&
			lastElement.Close == 0 {
			padded = padded[:len(padded)-1]
		}
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
	if desc {
		sort.Slice(k.Candles, func(i, j int) bool { return k.Candles[i].Time.After(k.Candles[j].Time) })
		return
	}
	sort.Slice(k.Candles, func(i, j int) bool { return k.Candles[i].Time.Before(k.Candles[j].Time) })
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
	case Raw:
		return "raw"
	case TenMilliseconds:
		return "tenmillisec"
	case TwentyMilliseconds:
		return "twentymillisec"
	case HundredMilliseconds:
		return "hundredmillisec"
	case TwoHundredAndFiftyMilliseconds:
		return "twohundredfiftymillisec"
	case ThousandMilliseconds:
		return "thousandmillisec"
	case TenSecond:
		return "tensec"
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
	case FiveDay:
		return "fiveday"
	case FifteenDay:
		return "fifteenday"
	case OneWeek:
		return "oneweek"
	case TwoWeek:
		return "twoweek"
	case OneMonth:
		return "onemonth"
	case ThreeMonth:
		return "threemonth"
	case SixMonth:
		return "sixmonth"
	case OneYear:
		return "oneyear"
	default:
		return "notfound"
	}
}

// TotalCandlesPerInterval returns the total number of candle intervals between the start and end date
func TotalCandlesPerInterval(start, end time.Time, interval Interval) uint64 {
	if interval <= 0 {
		return 0
	}

	if start.After(end) {
		return 0
	}

	window := end.Sub(start)
	return uint64(window) / uint64(interval) //nolint:gosec // No overflow risk
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
		// If this check does not pass, this candle has zero values or is padding.
		// It has nothing to apply to the new interval candle as it will distort
		// candle data.
		if k.Candles[x].Open != 0 &&
			k.Candles[x].High != 0 &&
			k.Candles[x].Low != 0 &&
			k.Candles[x].Close != 0 &&
			k.Candles[x].Volume != 0 {
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
			candles[target].Close = k.Candles[x].Close
		}

		if (x+1)%oldIntervalsPerNewCandle == 0 {
			// Note: Below checks the length of the proceeding slice so we can
			// break instantly if we cannot make an entire candle. e.g. 60 min
			// candles in an hour candle and we have 59 minute candles left.
			// This entire procession is cleaved.
			if len(k.Candles[x:])-1 < oldIntervalsPerNewCandle {
				break
			}
			target++
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
func CalculateCandleDateRanges(start, end time.Time, interval Interval, limit uint64) (*IntervalRangeHolder, error) {
	if err := common.StartEndTimeCheck(start, end); err != nil && !errors.Is(err, common.ErrStartAfterTimeNow) {
		return nil, err
	}
	if interval <= 0 {
		return nil, ErrInvalidInterval
	}

	start = start.Round(interval.Duration())
	end = end.Round(interval.Duration())

	count := uint64(end.Sub(start) / interval.Duration()) //nolint:gosec // No overflow risk
	if count == 0 {
		return nil, common.ErrStartEqualsEnd
	}

	intervals := make([]IntervalData, 0, count)
	for iStart := start; iStart.Before(end); iStart = iStart.Add(interval.Duration()) {
		intervals = append(intervals, IntervalData{
			Start: CreateIntervalTime(iStart),
			End:   CreateIntervalTime(iStart.Add(interval.Duration())),
		})
	}

	if limit == 0 {
		limit = count
	}

	h := &IntervalRangeHolder{
		Start: CreateIntervalTime(start),
		End:   CreateIntervalTime(end),
		Limit: limit,
	}

	for _, b := range common.Batch(intervals, int(limit)) { //nolint:gosec // Ignore this warning as Batch requires int
		h.Ranges = append(h.Ranges, IntervalRange{
			Start:     b[0].Start,
			End:       b[len(b)-1].End,
			Intervals: b,
		})
	}

	return h, nil
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
func (h *IntervalRangeHolder) SetHasDataFromCandles(incoming []Candle) error {
	var offset int
	for x := range h.Ranges {
		for y := range h.Ranges[x].Intervals {
			if offset >= len(incoming) {
				return nil
			}
			if !h.Ranges[x].Intervals[y].Start.Time.Equal(incoming[offset].Time) {
				return fmt.Errorf("%w '%v' expected '%v'", errInvalidPeriod, incoming[offset].Time.UTC(), h.Ranges[x].Intervals[y].Start.Time.UTC())
			}
			if incoming[offset].Low <= 0 && incoming[offset].High <= 0 &&
				incoming[offset].Close <= 0 && incoming[offset].Open <= 0 &&
				incoming[offset].Volume <= 0 {
				h.Ranges[x].Intervals[y].HasData = false
			} else {
				h.Ranges[x].Intervals[y].HasData = true
			}
			offset++
		}
	}
	return nil
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
		start.Format(time.DateTime),
		end.Format(time.DateTime))
}

// CreateIntervalTime is a simple helper function to set the time twice
func CreateIntervalTime(tt time.Time) IntervalTime {
	return IntervalTime{Time: tt, Ticks: tt.Unix()}
}

// Equal allows for easier unix comparison
func (i *IntervalTime) Equal(tt time.Time) bool {
	return tt.Unix() == i.Ticks
}

// EqualSource checks whether two sets of candles
// come from the same data source
func (k *Item) EqualSource(i *Item) error {
	if k == nil || i == nil {
		return common.ErrNilPointer
	}
	if k.Exchange != i.Exchange ||
		k.Asset != i.Asset ||
		!k.Pair.Equal(i.Pair) {
		return fmt.Errorf("%v %v %v %w %v %v %v", k.Exchange, k.Asset, k.Pair, ErrItemNotEqual, i.Exchange, i.Asset, i.Pair)
	}
	if !k.UnderlyingPair.IsEmpty() && !i.UnderlyingPair.IsEmpty() && !k.UnderlyingPair.Equal(i.UnderlyingPair) {
		return fmt.Errorf("%w %v %v", ErrItemUnderlyingNotEqual, k.UnderlyingPair, i.UnderlyingPair)
	}
	return nil
}

// DeployExchangeIntervals aligns and stores supported intervals for an exchange
// for future matching.
func DeployExchangeIntervals(enabled ...IntervalCapacity) ExchangeIntervals {
	sort.Slice(enabled, func(i, j int) bool { return enabled[i].Interval < enabled[j].Interval })

	supported := make(map[Interval]uint64)
	for x := range enabled {
		supported[enabled[x].Interval] = enabled[x].Capacity
	}
	return ExchangeIntervals{supported: supported, aligned: enabled}
}

// ExchangeSupported returns if the exchange directly supports the interval. In
// future this might be able to be deprecated because we can construct custom
// intervals from the supported list.
func (e *ExchangeIntervals) ExchangeSupported(in Interval) bool {
	_, ok := e.supported[in]
	return ok
}

// Construct fetches supported interval that can construct the required interval
// e.g. 1 hour interval candles can be made from 2 * 30 minute interval candles.
func (e *ExchangeIntervals) Construct(required Interval) (Interval, error) {
	if required <= 0 {
		return 0, ErrInvalidInterval
	}

	if _, ok := e.supported[required]; ok {
		// Directly supported by exchange can return.
		return required, nil
	}

	for x := len(e.aligned) - 1; x > -1; x-- {
		if e.aligned[x].Interval < required && required%e.aligned[x].Interval == 0 {
			// Indirectly supported by exchange. Can generate required candle
			// from this lower time frame supported candle.
			return e.aligned[x].Interval, nil
		}
	}
	return 0, ErrCannotConstructInterval
}

// GetIntervalResultLimit returns the maximum amount of candles that can be
// returned for a specific interval. If the individual interval limit is not set,
// it will be ignored and the global result limit will be returned.
func (e *ExchangeCapabilitiesEnabled) GetIntervalResultLimit(interval Interval) (uint64, error) {
	if e == nil {
		return 0, errExchangeCapabilitiesEnabledIsNil
	}

	val, ok := e.Intervals.supported[interval]
	if !ok {
		return 0, fmt.Errorf("[%s] %w", interval, errIntervalNotSupported)
	}

	if val > 0 {
		return val, nil
	}

	if e.GlobalResultLimit == 0 {
		return 0, fmt.Errorf("%w there is no global result limit set", errCannotFetchIntervalLimit)
	}

	return e.GlobalResultLimit, nil
}
