package data

import (
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
)

const (
	DataTypeCandle portfolio.DataType = iota
	DataTypeTick
)

type Candle struct {
	event.Event
	Open   float64
	Close  float64
	Low    float64
	High   float64
	Volume float64
}
type Tick struct {
	event.Event
	Bid float64
	Ask float64
}

type Orderbook struct {
	event.Event
	Bids []float64
	Asks []float64
}

type Data struct {
	latest portfolio.DataEventHandler
	stream []portfolio.DataEventHandler

	offset int
}
