package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Order struct {
	Event
	id           int
	orderType    order.Type
	orderSide    order.Side
	status       order.Status
	amount       float64
	amountFilled float64
	avgFillPrice float64
	limitPrice   float64

	fillTime time.Time
	fee      float64
	cost     float64
}

type OrderEvent interface {
	EventHandler

	Direction() order.Side
	SetOrderType(orderType order.Type)
	GetOrderType() (orderType order.Type)

	Amount() float64
	SetAmount(float64)

	ID() int
	SetID(int)
	Status() order.Status

	Price() float64
	Fee() float64
	Cost() float64
	Value() float64
	NetValue() float64
}
