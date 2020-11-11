package size

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/internalordermanager"
	order2 "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (s *Size) SizeOrder(o internalordermanager.OrderEvent, _ interfaces.DataEventHandler, availableFunds, feeRate float64) (*order.Order, error) {
	retOrder := o.(*order.Order)

	if (s.DefaultSize == 0) || (s.MaxSize == 0) {
		return nil, errors.New("no defaultSize or defaultValue set")
	}

	switch retOrder.GetDirection() {
	case order2.Buy:
		retOrder.SetAmount(s.calculateSize(retOrder.Price, availableFunds, feeRate))
	case order2.Sell:
		retOrder.SetAmount(s.calculateSize(retOrder.Price, availableFunds, feeRate))
	}
	return retOrder, nil
}

func (s *Size) calculateSize(price float64, availableFunds, feeRate float64) float64 {
	if availableFunds <= 0 {
		return 0
	}
	if availableFunds/price > s.DefaultSize {
		amount := s.DefaultSize
		fee := amount * feeRate * price
		amount -= fee
		return amount
	}
	amount := availableFunds / price
	fee := amount * feeRate * price
	amount -= fee
	return amount
}
