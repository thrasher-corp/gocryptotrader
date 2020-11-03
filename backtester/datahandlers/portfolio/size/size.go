package size

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"
	order2 "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (s *Size) SizeOrder(o orders.OrderEvent, _ interfaces.DataEventHandler) (*order.Order, error) {
	retOrder := o.(*order.Order)

	if (s.DefaultSize == 0) || (s.MaxSize == 0) {
		return nil, errors.New("no defaultSize or defaultValue set")
	}

	switch retOrder.GetDirection() {
	case order2.Buy:
		retOrder.SetAmount(s.setDefaultSize(retOrder.Price))
	case order2.Sell:
		retOrder.SetAmount(s.setDefaultSize(retOrder.Price))
	}
	return retOrder, nil
}

func (s *Size) setDefaultSize(price float64) float64 {
	if (price / s.DefaultSize) > s.MaxSize {
		return price / s.MaxSize
	}
	return price / s.DefaultSize
}
