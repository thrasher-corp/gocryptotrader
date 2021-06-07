package kline

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Consts here define basic time intervals
const (
	FifteenSecond = Interval(15 * time.Second)
	OneMin        = Interval(time.Minute)
	ThreeMin      = 3 * OneMin
	FiveMin       = 5 * OneMin
	TenMin        = 10 * OneMin
	FifteenMin    = 15 * OneMin
	ThirtyMin     = 30 * OneMin
	OneHour       = Interval(time.Hour)
	TwoHour       = 2 * OneHour
	FourHour      = 4 * OneHour
	SixHour       = 6 * OneHour
	EightHour     = 8 * OneHour
	TwelveHour    = 12 * OneHour
	OneDay        = 24 * OneHour
	ThreeDay      = 3 * OneDay
	SevenDay      = 7 * OneDay
	FifteenDay    = 15 * OneDay
	OneWeek       = 7 * OneDay
	TwoWeek       = 2 * OneWeek
	OneMonth      = 31 * OneDay
	OneYear       = 365 * OneDay
)

const (
	// ErrRequestExceedsExchangeLimits locale for exceeding rate limits message
	ErrRequestExceedsExchangeLimits = "requested data would exceed exchange limits please lower range or use GetHistoricCandlesEx"
)

var (
	// ErrMissingCandleData is an error for missing candle data
	ErrMissingCandleData = errors.New("missing candle data")
	// ErrUnsetInterval is an error for date range calculation
	ErrUnsetInterval = errors.New("cannot calculate range, interval unset")
	// ErrUnsupportedInterval returns when the provided interval is not supported by an exchange
	ErrUnsupportedInterval = errors.New("interval unsupported by exchange")

	// SupportedIntervals is a list of all supported intervals
	SupportedIntervals = []Interval{
		FifteenSecond,
		OneMin,
		ThreeMin,
		FiveMin,
		TenMin,
		FifteenMin,
		ThirtyMin,
		OneHour,
		TwoHour,
		FourHour,
		SixHour,
		EightHour,
		TwelveHour,
		OneDay,
		ThreeDay,
		SevenDay,
		FifteenDay,
		OneWeek,
		TwoWeek,
		OneMonth,
		OneYear,
	}
)

// Item holds all the relevant information for internal kline elements
type Item struct {
	Exchange string
	Pair     currency.Pair
	Asset    asset.Item
	Interval Interval
	Candles  []Candle
}

// Candle holds historic rate information.
type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// ByDate allows for sorting candle entries by date
type ByDate []Candle

func (b ByDate) Len() int {
	return len(b)
}

func (b ByDate) Less(i, j int) bool {
	return b[i].Time.Before(b[j].Time)
}

func (b ByDate) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// ExchangeCapabilitiesSupported all kline related exchange supported options
type ExchangeCapabilitiesSupported struct {
	Intervals  bool
	DateRanges bool
}

// ExchangeCapabilitiesEnabled all kline related exchange enabled options
type ExchangeCapabilitiesEnabled struct {
	Intervals   map[string]bool `json:"intervals,omitempty"`
	ResultLimit uint32
}

// Interval type for kline Interval usage
type Interval time.Duration

// ErrorKline struct to hold kline interval errors
type ErrorKline struct {
	Asset    asset.Item
	Pair     currency.Pair
	Interval Interval
	Err      error
}

// Error returns short interval unsupported message
func (k *ErrorKline) Error() string {
	return k.Err.Error()
}

// Unwrap returns interval unsupported message
func (k *ErrorKline) Unwrap() error {
	return k.Err
}

// IntervalRangeHolder holds the entire range of intervals
// and the start end dates of everything
type IntervalRangeHolder struct {
	Start  IntervalTime
	End    IntervalTime
	Ranges []IntervalRange
}

// IntervalRange is a subset of candles based on exchange API request limits
type IntervalRange struct {
	Start     IntervalTime
	End       IntervalTime
	Intervals []IntervalData
}

// IntervalData is used to monitor which candles contain data
// to determine if any data is missing
type IntervalData struct {
	Start   IntervalTime
	End     IntervalTime
	HasData bool
}

// IntervalTime benchmarks demonstrate, see
// BenchmarkJustifyIntervalTimeStoringUnixValues1 &&
// BenchmarkJustifyIntervalTimeStoringUnixValues2
type IntervalTime struct {
	Time  time.Time
	Ticks int64
}
