package backtest

import (
	"errors"
	"math"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (r *Risk) EvaluateOrder(order OrderEvent) (*Order, error) {
	return nil, nil
}

func (s *Size) SizeOrder(orderevent OrderEvent, data DataEvent, pf PortfolioHandler) (*Order, error) {
	if (s.DefaultSize == 0) || (s.DefaultValue == 0) {
		return nil, errors.New("no DefaultSize or DefaultValue set")
	}

	o := orderevent.(*Order)
	switch o.Direction() {
	case order.Buy:
		o.SetAmount(s.setDefaultSize(data.Price()))
	case order.Sell:
		o.SetAmount(s.setDefaultSize(data.Price()))
	default:
		if _, ok := pf.IsInvested(); !ok {
			return o, errors.New("no position in portfolio")
		}
		if pos, ok := pf.IsLong(); ok {
			o.SetAmount(pos.Amount)
		}
		if pos, ok := pf.IsShort(); ok {
			o.SetAmount(pos.Amount * -1)
		}
	}

	return o, nil
}

func (s *Size) setDefaultSize(price float64) float64 {
	if (s.DefaultSize * price) > s.DefaultValue {
		return math.Floor(s.DefaultValue / price)
	}
	return s.DefaultSize
}
