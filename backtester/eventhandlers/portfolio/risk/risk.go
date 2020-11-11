package risk

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/internalordermanager"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	order2 "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// TODO implement risk manager
func (r *Risk) EvaluateOrder(o internalordermanager.OrderEvent, _ portfolio.DataEventHandler, currenntPosition positions.Positions, allPositions map[string]map[asset.Item]map[currency.Pair]positions.Positions) (*order.Order, error) {
	retOrder := o.(*order.Order)

	if o.IsLeveraged() {
		ratio := existingLeverageRatio()

		if ratio > r.MaxLeverageRatio[o.GetExchange()][o.GetAssetType()][o.Pair()] {
			return nil, nil
		}
		if retOrder.GetLeverage() > r.MaxLeverageRate[o.GetExchange()][o.GetAssetType()][o.Pair()] {
			return nil, nil
		}

	}
	return retOrder, nil
}

func existingLeverageRatio() float64 {
	os, _ := engine.Bot.OrderManager.GetOrdersSnapshot(order2.AnyStatus)
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

func AreWeAllIn() {

}
