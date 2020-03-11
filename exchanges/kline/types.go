package kline

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Consts here define basic time intervals
const (
	OneMin     = time.Minute
	ThreeMin   = 3 * time.Minute
	FiveMin    = 5 * time.Minute
	FifteenMin = 15 * time.Minute
	ThirtyMin  = 30 * time.Minute
	OneHour    = 1 * time.Hour
	TwoHour    = 2 * time.Hour
	FourHour   = 4 * time.Hour
	SixHour    = 6 * time.Hour
	TwelveHour = 12 * time.Hour
	OneDay     = 24 * time.Hour
	ThreeDay   = 72 * time.Hour
	OneWeek    = 168 * time.Hour
)

// Item holds all the relevant information for internal kline elements
type Item struct {
	Exchange string
	Pair     currency.Pair
	Asset    asset.Item
	Interval time.Duration
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
