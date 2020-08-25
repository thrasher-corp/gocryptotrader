package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type DataEvent interface {
	EventHandler
	Price() float64
	Candle() Candle
}

type Event struct {
	time time.Time
}

type EventHandler interface {
	Time() time.Time
	SetTime(time.Time)
}

type SignalEvent interface {
	EventHandler

	Direction() order.Side
	SetOrderType(orderType order.Type)
	GetOrderType() (orderType order.Type)
}
