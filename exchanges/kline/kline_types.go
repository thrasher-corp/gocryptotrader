package kline

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Consts here define basic time intervals
const (
	Raw                            = Interval(-1)
	TenMilliseconds                = Interval(10 * time.Millisecond)
	TwentyMilliseconds             = 2 * TenMilliseconds
	HundredMilliseconds            = Interval(100 * time.Millisecond)
	TwoHundredAndFiftyMilliseconds = Interval(250 * time.Millisecond)
	ThousandMilliseconds           = 10 * HundredMilliseconds
	TenSecond                      = Interval(10 * time.Second)
	FifteenSecond                  = Interval(15 * time.Second)
	ThirtySecond                   = 2 * FifteenSecond
	OneMin                         = Interval(time.Minute)
	ThreeMin                       = 3 * OneMin
	FiveMin                        = 5 * OneMin
	TenMin                         = 10 * OneMin
	FifteenMin                     = 15 * OneMin
	ThirtyMin                      = 30 * OneMin
	OneHour                        = Interval(time.Hour)
	TwoHour                        = 2 * OneHour
	ThreeHour                      = 3 * OneHour
	FourHour                       = 4 * OneHour
	SixHour                        = 6 * OneHour
	SevenHour                      = 7 * OneHour
	EightHour                      = 8 * OneHour
	TwelveHour                     = 12 * OneHour
	OneDay                         = 24 * OneHour
	TwoDay                         = 2 * OneDay
	ThreeDay                       = 3 * OneDay
	SevenDay                       = 7 * OneDay
	FifteenDay                     = 15 * OneDay
	OneWeek                        = 7 * OneDay
	TwoWeek                        = 2 * OneWeek
	ThreeWeek                      = 3 * OneWeek
	OneMonth                       = 30 * OneDay
	ThreeMonth                     = 90 * OneDay
	SixMonth                       = 2 * ThreeMonth
	NineMonth                      = 3 * ThreeMonth
	OneYear                        = 365 * OneDay
	FiveDay                        = 5 * OneDay
)

var (
	// ErrRequestExceedsExchangeLimits locale for exceeding rate limits message
	ErrRequestExceedsExchangeLimits = errors.New("request will exceed exchange limits, please reduce start-end time window or use GetHistoricCandlesExtended")
	// ErrUnsupportedInterval returns when the provided interval is not supported by an exchange
	ErrUnsupportedInterval = errors.New("interval unsupported by exchange")
	// ErrCanOnlyUpscaleCandles returns when attempting to upscale candles
	ErrCanOnlyUpscaleCandles = errors.New("interval must be a longer duration to scale")
	// ErrWholeNumberScaling returns when old interval data cannot neatly fit into new interval size
	ErrWholeNumberScaling = errors.New("old interval must scale properly into new candle")
	// ErrNotFoundAtTime returned when looking up a candle at a specific time
	ErrNotFoundAtTime = errors.New("candle not found at time")
	// ErrItemNotEqual returns when comparison between two kline items fail
	ErrItemNotEqual = errors.New("kline item not equal")
	// ErrItemUnderlyingNotEqual returns when the underlying pair is not equal
	ErrItemUnderlyingNotEqual = errors.New("kline item underlying pair not equal")
	// ErrValidatingParams defines an error when the kline params are either not
	// enabled or are invalid.
	ErrValidatingParams = errors.New("kline param(s) are invalid")
	// ErrInvalidInterval defines when an interval is invalid e.g. interval <= 0
	ErrInvalidInterval = errors.New("invalid/unset interval")
	// ErrCannotConstructInterval defines an error when an interval cannot be
	// constructed from a list of support intervals.
	ErrCannotConstructInterval = errors.New("cannot construct required interval from supported intervals")
	// ErrInsufficientCandleData defines an error when you have a candle that
	// requires multiple candles to generate.
	ErrInsufficientCandleData = errors.New("insufficient candle data to generate new candle")
	// ErrRequestExceedsMaxLookback defines an error for when you cannot look
	// back further than what is allowed.
	ErrRequestExceedsMaxLookback = errors.New("the requested time window exceeds the maximum lookback period available in the historical data, please reduce window between start and end date of your request")

	errInsufficientTradeData            = errors.New("insufficient trade data")
	errCandleDataNotPadded              = errors.New("candle data not padded")
	errCannotEstablishTimeWindow        = errors.New("cannot establish time window")
	errNilKline                         = errors.New("kline item is nil")
	errExchangeCapabilitiesEnabledIsNil = errors.New("exchange capabilities enabled is nil")
	errCannotFetchIntervalLimit         = errors.New("cannot fetch interval limit")
	errIntervalNotSupported             = errors.New("interval not supported")
	errCandleOpenTimeIsNotUTCAligned    = errors.New("candle open time is not UTC aligned")

	oneYearDurationInNano = float64(OneYear.Duration().Nanoseconds())

	// SupportedIntervals is a list of all supported intervals
	SupportedIntervals = []Interval{
		HundredMilliseconds,
		ThousandMilliseconds,
		TenSecond,
		FifteenSecond,
		OneMin,
		ThreeMin,
		FiveMin,
		TenMin,
		FifteenMin,
		ThirtyMin,
		OneHour,
		TwoHour,
		ThreeHour,
		FourHour,
		SixHour,
		SevenHour,
		EightHour,
		TwelveHour,
		OneDay,
		ThreeDay,
		FiveDay,
		SevenDay,
		FifteenDay,
		OneWeek,
		TwoWeek,
		OneMonth,
		ThreeMonth,
		SixMonth,
		OneYear,
		ThreeMonth,
		SixMonth,
	}
)

// Item holds all the relevant information for internal kline elements
type Item struct {
	Exchange        string
	Pair            currency.Pair
	UnderlyingPair  currency.Pair
	Asset           asset.Item
	Interval        Interval
	Candles         []Candle
	SourceJobID     uuid.UUID
	ValidationJobID uuid.UUID
}

// Candle holds historic rate information.
type Candle struct {
	Time             time.Time
	Open             float64
	High             float64
	Low              float64
	Close            float64
	Volume           float64
	ValidationIssues string
}

// ExchangeCapabilitiesSupported all kline related exchange supported options
type ExchangeCapabilitiesSupported struct {
	Intervals  bool
	DateRanges bool
}

// ExchangeCapabilitiesEnabled all kline related exchange enabled options
type ExchangeCapabilitiesEnabled struct {
	// Intervals defines whether the exchange supports interval kline requests.
	Intervals ExchangeIntervals
	// GlobalResultLimit is the maximum amount of candles that can be returned
	// across all intervals. This is used to determine if a request will exceed
	// the exchange limits. Indivudal interval limits are stored in the
	// ExchangeIntervals struct. If this is set to 0, it will be ignored.
	GlobalResultLimit uint64
}

// ExchangeIntervals stores the supported intervals in an optimized lookup table
// with a supplementary aligned retrieval list
type ExchangeIntervals struct {
	supported map[Interval]uint64
	aligned   []IntervalCapacity
}

// Interval type for kline Interval usage
type Interval time.Duration

// IntervalRangeHolder holds the entire range of intervals
// and the start end dates of everything
type IntervalRangeHolder struct {
	Start  IntervalTime
	End    IntervalTime
	Ranges []IntervalRange
	Limit  uint64
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

// IntervalCapacity is used to store the interval and capacity for a candle return
type IntervalCapacity struct {
	Interval Interval
	Capacity uint64
}
