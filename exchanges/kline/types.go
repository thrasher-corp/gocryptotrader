package kline

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Consts here define basic time intervals
const (
	OneMin         = Interval(time.Minute)
	ThreeMin       = 3 * OneMin
	FiveMin        = 5 * OneMin
	TenMin         = 10 * OneMin
	FifteenMin     = 15 * OneMin
	ThirtyMin      = 30 * OneMin
	OneHour        = Interval(1 * time.Hour)
	TwoHour        = 2 * OneHour
	FourHour       = 4 * OneHour
	SixHour        = 6 * OneHour
	TwelveHour     = 12 * OneHour
	TwentyFourHour = 24 * OneHour
	OneDay         = TwentyFourHour
	ThreeDay       = 3 * OneDay
	SevenDay       = 7 * OneDay
	OneWeek        = 7 * OneDay
	TwoWeek        = 2 * OneWeek
)

// ErrUnsupportedInterval locacle for an unsupported interval
const ErrUnsupportedInterval = "%s interval unsupported by exchange"

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

// ExchangeCapabilities all kline related exchane supported options
type ExchangeCapabilities struct {
	SupportsIntervals bool
	Intervals         map[string]bool `json:"intervals,omitempty"`
	SupportsDateRange bool
	Limit             uint32
}

type Interval time.Duration

type ErrorKline struct {
	Interval Interval
}

func (k ErrorKline) Error() string {
	return fmt.Sprintf(ErrUnsupportedInterval, k.Interval.Short())
}

func (k *ErrorKline) Unwrap() error {
	return fmt.Errorf(ErrUnsupportedInterval, k.Interval)
}
