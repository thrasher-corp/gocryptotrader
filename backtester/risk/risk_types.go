package risk

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/orderbook"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

type RiskHandler interface {
	EvaluateOrder(orderbook.OrderEvent, datahandler.DataEventHandler, map[currency.Pair]positions.Positions) (*order.Order, error)
}

type Risk struct{}
