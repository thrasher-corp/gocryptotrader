package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type DataEvent interface {
	EventHandler
	Price() float64
}

type EventHandler interface {
	Time() time.Time
	SetTime(time.Time)
}

type DataHandler interface {
	Next() (DataEvent, bool)
	Stream() []DataEvent
	History() []DataEvent
	Latest() DataEvent

	Reset()
}


type SignalEvent interface {
	EventHandler

	Direction() order.Side
	SetOrderType(orderType order.Type)
	GetOrderType()(orderType order.Type)
}
