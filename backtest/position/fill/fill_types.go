package fill

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/event"
)

type Handler interface {
	event.Handler
	event.Direction

	Amount() float64
	SetAmount(float64)

	Price()  float64

	Fee()    float64
	Cost() float64
	Value() float64
	NetValue() float64
}

type Event struct {
	event.Handler
	event.Direction

	amount float64
	price  float64
	fee    float64
}
