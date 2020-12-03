package risk

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// EvaluateOrder goes through a standard list of evaluations to make to ensure that
// we are in a position to follow through with an order
func (r *Risk) EvaluateOrder(o order.OrderEvent, _ interfaces.DataEventHandler, latestHoldings []holdings.Holding) (*order.Order, error) {
	retOrder := o.(*order.Order)
	if o.IsLeveraged() {
		if !r.CanUseLeverage {
			return nil, errors.New("order says to use leverage, but its not allowed damnit")
		}
		ratio := existingLeverageRatio()

		if ratio > r.MaxLeverageRatio[o.GetExchange()][o.GetAssetType()][o.Pair()] {
			return nil, fmt.Errorf("leveraged ratio over maximum threshold for order %v", o)
		}
		if retOrder.GetLeverage() > r.MaxLeverageRate[o.GetExchange()][o.GetAssetType()][o.Pair()] {
			return nil, fmt.Errorf("leverage level over maximum for order %v", o)
		}
	}
	allIn := areWeAllInOnOneCurrency(o.Pair(), latestHoldings)
	if allIn {

	}
	ratios := assessHoldingsRatio(o.Pair(), latestHoldings)
	if len(ratios) == 0 {

	}
	return retOrder, nil
}

func existingLeverageRatio() float64 {
	os, _ := engine.Bot.OrderManager.GetOrdersSnapshot(gctorder.AnyStatus)
	if len(os) == 0 {
		return 0
	}
	var ordersWithLeverage float64
	for o := range os {
		if os[o].Leverage != "" {
			ordersWithLeverage++
		}
	}
	return ordersWithLeverage / float64(len(os))
}

func areWeAllInOnOneCurrency(c currency.Pair, h []holdings.Holding) bool {
	for i := range h {
		if !h[i].Pair.Equal(c) {
			return false
		}
	}
	return true
}

// add additional assessing rules, such as what the maximum ratio is allowed to be
func assessHoldingsRatio(c currency.Pair, h []holdings.Holding) map[currency.Pair]float64 {
	resp := make(map[currency.Pair]float64)
	for i := range h {
		if h[i].Pair.Equal(c) {
			resp[c] += h[i].PositionsSize
		}
	}
	return resp
}
