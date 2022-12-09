package kline

import (
	"errors"
	"fmt"
	"strings"
	"time"
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
	TwoDay        = 2 * OneDay
	ThreeDay      = 3 * OneDay
	SevenDay      = 7 * OneDay
	FifteenDay    = 15 * OneDay
	OneWeek       = SevenDay
	TwoWeek       = 2 * OneWeek
	OneMonth      = 31 * OneDay
	ThreeMonth    = 3 * OneMonth
	SixMonth      = 6 * OneMonth
	OneYear       = 365 * OneDay
)

// supportedIntervals is a list of all supported intervals
var supportedIntervals = map[int64]Interval{
	int64(FifteenSecond): FifteenSecond,
	int64(OneMin):        OneMin,
	int64(ThreeMin):      ThreeMin,
	int64(FiveMin):       FiveMin,
	int64(TenMin):        TenMin,
	int64(FifteenMin):    FifteenMin,
	int64(ThirtyMin):     ThirtyMin,
	int64(OneHour):       OneHour,
	int64(TwoHour):       TwoHour,
	int64(FourHour):      FourHour,
	int64(SixHour):       SixHour,
	int64(EightHour):     EightHour,
	int64(TwelveHour):    TwelveHour,
	int64(OneDay):        OneDay,
	int64(ThreeDay):      ThreeDay,
	int64(FifteenDay):    FifteenDay,
	int64(OneWeek):       OneWeek,
	int64(TwoWeek):       TwoWeek,
	int64(OneMonth):      OneMonth,
	int64(OneYear):       OneYear,
}

var (
	// ErrUnsetInterval is an error for date range calculation
	ErrUnsetInterval = errors.New("cannot calculate range, interval unset")
	// ErrUnsupportedInterval returns when the provided interval is not supported by an exchange
	ErrUnsupportedInterval = errors.New("interval unsupported by exchange")
	// ErrInvalidIntervalNumber defines an error when it is unset
	ErrInvalidIntervalNumber = errors.New("invalid interval must be greater than zero")
)

// NewInterval returns a new interval derived from a nanosecond integer. This
// checks against supported list if a specific custom interval is *not* being
// generated.
func NewInterval(ns int64, custom bool) (Interval, error) {
	if ns <= 0 {
		return 0, ErrInvalidIntervalNumber
	}
	if custom {
		return Interval(ns), nil
	}
	interval, ok := supportedIntervals[ns]
	if !ok {
		return 0, fmt.Errorf("[%d] %w", ns, ErrUnsupportedInterval)
	}
	return interval, nil
}

// GetSupportedIntervals returns the list of supported intervals
func GetSupportedIntervals() []Interval {
	supported := make([]Interval, len(supportedIntervals))
	var target int
	for _, interval := range supportedIntervals {
		supported[target] = interval
		target++
	}
	return supported
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
