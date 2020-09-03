package backtest

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Order struct {
	Event
	id        int
	Direction order.Side
	Status    order.Status
	Price     float64
	Amount    float64
	OrderType order.Type
	limit     float64
}

type OrderBook struct {
	counter int
	orders  []OrderEvent
	history []OrderEvent

	m sync.Mutex
}
