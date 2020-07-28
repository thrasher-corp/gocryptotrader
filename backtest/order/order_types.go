package order

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Handler interface {
	event.Handler
	event.Direction

	ID() uint64
	SetID(uint64)

	Amount() float64
	SetAmount(float64)
}

type Order struct {
	event.Event
	id                 uint64
	Type               order.Type
	Status             order.Status
	Direction          event.Directions
	Asset              asset.Item
	amount             float64
	amountFilled       float64
	AverageFilledPrice float64
	Price              float64
	StopPrice          float64
}
