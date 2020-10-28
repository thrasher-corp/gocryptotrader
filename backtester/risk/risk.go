package risk

import (
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/orderbook"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// TODO implement risk manager
func (r *Risk) EvaluateOrder(o orderbook.OrderEvent, _ portfolio.DataEventHandler, _ map[currency.Pair]positions.Positions) (*order.Order, error) {
	retOrder := o.(*order.Order)

	if o.IsLeveraged() {
		// handle risk
	}
	return retOrder, nil
}
