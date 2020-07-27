package data

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type Handler interface {
	Load(in []kline.Candle) error
	Reset()

	Stream
}

type Stream interface {
	Next() (Event, bool)
	Stream() []Event
	History() []Event
	Latest(string) Event
	List(string) []Event
}

type Data struct {
	latest  map[string]Event
	list    map[string][]Event
	stream  []Event
	history []Event
}

type Event struct {
	event.Handler
	price float64
}

type Tick struct {
	event.Handler
	Bid    float64
	Ask    float64
	BidVol float64
	AskVol float64
}

type TickEvent struct {
	Event
}
