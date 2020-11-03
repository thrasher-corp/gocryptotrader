package orders

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/backtester/direction"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// OrderEvent
type OrderEvent interface {
	interfaces.EventHandler
	direction.Directioner

	SetAmount(float64)
	GetAmount() float64
	IsOrder() bool

	GetStatus() order.Status
	SetID(id int)
	GetID() int
	GetLimit() float64
	IsLeveraged() bool
}

type Orders struct {
	Counter int
	Orders  []OrderEvent
	History []OrderEvent

	M sync.Mutex
}
