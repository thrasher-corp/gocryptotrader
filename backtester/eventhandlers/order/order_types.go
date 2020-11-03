package order

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Order struct {
	event.Event
	ID        int
	Direction order.Side
	Status    order.Status
	Price     float64
	Amount    float64
	OrderType order.Type
	Limit     float64
	Leverage  float64
}
