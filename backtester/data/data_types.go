package data

import (
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
)

const (
	DataTypeCandle portfolio.DataType = iota
	DataTypeTick
)

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
