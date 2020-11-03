package kline

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
)

type Kline struct {
	event.Event
	Open   float64
	Close  float64
	Low    float64
	High   float64
	Volume float64
}
