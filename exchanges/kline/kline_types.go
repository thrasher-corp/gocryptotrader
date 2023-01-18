package kline

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrCanOnlyDownscaleCandles returns when attempting to upscale candles
	ErrCanOnlyDownscaleCandles = errors.New("interval must be a longer duration to scale")
)

var (
	// ErrRequestExceedsExchangeLimits locale for exceeding rate limits message
	ErrRequestExceedsExchangeLimits = errors.New("request will exceed exchange limits, please reduce start-end time window or use GetHistoricCandlesExtended")
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

	errInsufficientTradeData     = errors.New("insufficient trade data")
	errCandleDataNotPadded       = errors.New("candle data not padded")
	errCannotEstablishTimeWindow = errors.New("cannot establish time window")
	errNilKline                  = errors.New("kline item is nil")

	oneYearDurationInNano = float64(OneYear.Duration().Nanoseconds())
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
	Intervals   ExchangeIntervals
	ResultLimit uint32
}

// ExchangeIntervals stores the supported intervals in an optimized lookup table
// with a supplementary aligned retrieval list
type ExchangeIntervals struct {
	supported map[Interval]bool
	aligned   []Interval
}

// Interval type for kline Interval usage
type Interval time.Duration

// IntervalRangeHolder holds the entire range of intervals
// and the start end dates of everything
type IntervalRangeHolder struct {
	Start  IntervalTime
	End    IntervalTime
	Ranges []IntervalRange
	Limit  int
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
