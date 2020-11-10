package risk

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
)

// TODO implement risk manager
func (r *Risk) EvaluateOrder(o orders.OrderEvent, _ portfolio.DataEventHandler, _ positions.Positions) (*order.Order, error) {
	retOrder := o.(*order.Order)

	if o.IsLeveraged() {
		// handle risk
	}
	return retOrder, nil
}
