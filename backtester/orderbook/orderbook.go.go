package orderbook

import (
	"sync"
)

type OrderBook struct {
	Counter int
	Orders  []OrderEvent
	History []OrderEvent

	M sync.Mutex
}
