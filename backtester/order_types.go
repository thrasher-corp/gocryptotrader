package backtest

import "github.com/thrasher-corp/gocryptotrader/exchanges/order"

type OrderEvent interface {
	EventHandler

	Direction() order.Side
	SetOrderType(orderType order.Type)
	GetOrderType()(orderType order.Type)

	Amount() int64
	SetAmount(int64)

	ID() int
	SetID(int)
	Status() order.Status

	Price() float64
	Fee() float64
	Cost() float64
	Value() float64
	NetValue() float64
}
