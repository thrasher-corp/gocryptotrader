package size

import (
	"errors"
	"math"

	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/orderbook"
	"github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	order2 "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// TODO implement risk manager
func (s *Size) SizeOrder(o orderbook.OrderEvent, _ datahandler.DataEventHandler, _ portfolio.PortfolioHandler) (*order.Order, error) {
	retOrder := o.(*order.Order)

	if (s.DefaultSize == 0) || (s.DefaultValue == 0) {
		return nil, errors.New("no defaultSize or defaultValue set")
	}

	switch retOrder.GetDirection() {
	case order2.Buy:
	case order2.Sell:
		retOrder.SetAmount(s.setDefaultSize(retOrder.Price))
	}
	return retOrder, nil
}

func (s *Size) setDefaultSize(price float64) float64 {
	if (s.DefaultSize * price) > s.DefaultValue {
		return math.Floor(s.DefaultValue / price)
	}
	return s.DefaultSize
}
