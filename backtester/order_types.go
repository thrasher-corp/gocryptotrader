package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Order struct {
	Event
	Direction order.Side
	Amount    float64
	OrderType order.Type
	Limit     float64
}
