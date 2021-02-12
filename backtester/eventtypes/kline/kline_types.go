package kline

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
)

// Kline holds kline data and an event to be processed as
// a common.DataEventHandler type
type Kline struct {
	event.Base
	Open   float64
	Close  float64
	Low    float64
	High   float64
	Volume float64
}
