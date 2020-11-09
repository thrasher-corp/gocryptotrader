package risk

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
)

type RiskHandler interface {
	EvaluateOrder(orders.OrderEvent, interfaces.DataEventHandler, positions.Positions) (*order.Order, error)
}

type Risk struct{}
