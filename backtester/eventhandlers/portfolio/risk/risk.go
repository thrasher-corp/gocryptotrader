package risk

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/internalordermanager"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// EvaluateOrder goes through a standard list of evaluations to make to ensure that
// we are in a position to follow through with an order
func (r *Risk) EvaluateOrder(o internalordermanager.OrderEvent, _ interfaces.DataEventHandler, _ positions.Positions, allPositions map[string]map[asset.Item]map[currency.Pair]positions.Positions) (*order.Order, error) {
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
	err := areWeAllIn(o)
	if err != nil {
		return nil, err
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

func areWeAllIn(o internalordermanager.OrderEvent) error {
	os, _ := engine.Bot.OrderManager.GetOrdersSnapshot(gctorder.AnyStatus)
	if len(os) == 0 {
		return nil
	}
	// in this setion of code, we'd want to calculate the total value of holdins per currency
	// then once we know how much everything is worth, we can get a ratio and check it against
	// our limits
	return nil
}
