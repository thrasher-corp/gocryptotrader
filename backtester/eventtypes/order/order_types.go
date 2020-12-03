package order

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Order struct {
	event.Event
	ID        string
	Direction order.Side
	Status    order.Status
	Price     float64
	Amount    float64
	OrderType order.Type
	Limit     float64
	Leverage  float64
}

// OrderEvent
type OrderEvent interface {
	interfaces.EventHandler
	interfaces.Directioner

	SetAmount(float64)
	GetAmount() float64
	IsOrder() bool
	GetStatus() order.Status
	SetID(id string)
	GetID() string
	GetLimit() float64
	IsLeveraged() bool
}
