package risk

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// EvaluateOrder goes through a standard list of evaluations to make to ensure that
// we are in a position to follow through with an order
func (r *Risk) EvaluateOrder(o order.Event, latestHoldings []holdings.Holding, s compliance.Snapshot) (*order.Order, error) {
	if o == nil || latestHoldings == nil {
		return nil, common.ErrNilArguments
	}
	retOrder := o.(*order.Order)
	ex := o.GetExchange()
	a := o.GetAssetType()
	p := o.Pair()
	lookup, ok := r.CurrencySettings[ex][a][p]
	if !ok {
		return nil, fmt.Errorf("%v %v %v %w", ex, a, p, errNoCurrencySettings)
	}

	if o.IsLeveraged() {
		if !r.CanUseLeverage {
			return nil, errLeverageNotAllowed
		}
		ratio := existingLeverageRatio(s)
		if ratio > lookup.MaximumOrdersWithLeverageRatio && lookup.MaximumOrdersWithLeverageRatio > 0 {
			return nil, fmt.Errorf("proceeding with the order would put maximum orders using leverage ratio beyond its limit of %f to %f and %w", lookup.MaximumOrdersWithLeverageRatio, ratio, errCannotPlaceLeverageOrder)
		}
		if retOrder.GetLeverage() > lookup.MaxLeverageRate && lookup.MaxLeverageRate > 0 {
			return nil, fmt.Errorf("proceeding with the order would put leverage rate beyond its limit of %f to %f and %w", lookup.MaxLeverageRate, retOrder.GetLeverage(), errCannotPlaceLeverageOrder)
		}
	}
	if len(latestHoldings) > 1 {
		ratio := assessHoldingsRatio(o.Pair(), latestHoldings)
		if lookup.MaximumHoldingRatio > 0 && ratio != 1 && ratio > lookup.MaximumHoldingRatio {
			return nil, fmt.Errorf("order would exceed maximum holding ratio of %f to %f for %v %v %v. %w", lookup.MaximumHoldingRatio, ratio, ex, a, p, errCannotPlaceLeverageOrder)
		}
	}
	return retOrder, nil
}

// existingLeverageRatio compares orders with leverage to the total number of orders
// a proof of concept to demonstrate risk manager's ability to prevent an order from being placed
// when an order exceeds a config setting
func existingLeverageRatio(s compliance.Snapshot) float64 {
	if len(s.Orders) == 0 {
		return 0
	}
	var ordersWithLeverage float64
	for o := range s.Orders {
		if s.Orders[o].Leverage != 0 {
			ordersWithLeverage++
		}
	}
	return ordersWithLeverage / float64(len(s.Orders))
}

func assessHoldingsRatio(c currency.Pair, h []holdings.Holding) float64 {
	resp := make(map[currency.Pair]float64)
	totalPosition := 0.0
	for i := range h {
		resp[h[i].Pair] += h[i].PositionsValue
		totalPosition += h[i].PositionsValue
	}

	if totalPosition == 0 {
		return 0
	}
	ratio := resp[c] / totalPosition

	return ratio
}
