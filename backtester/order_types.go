package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Order struct {
	Event
	id           int
	orderType    order.Type
	status       order.Status
	amount       int64
	amountFilled int64
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
