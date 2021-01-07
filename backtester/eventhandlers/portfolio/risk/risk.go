package risk

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// EvaluateOrder goes through a standard list of evaluations to make to ensure that
// we are in a position to follow through with an order
func (r *Risk) EvaluateOrder(o order.OrderEvent, latestHoldings []holdings.Holding, s compliance.Snapshot) (*order.Order, error) {
	if o == nil || latestHoldings == nil {
		return nil, errors.New("received nil argument(s)")
	}
	retOrder := o.(*order.Order)
	if o.IsLeveraged() {
		if !r.CanUseLeverage {
			return nil, errors.New("order says to use leverage, but it is not allowed")
		}
		ratio := existingLeverageRatio(s)
		lookupRatio := r.MaxLeverageRatio[o.GetExchange()][o.GetAssetType()][o.Pair()]
		if ratio > lookupRatio && lookupRatio > 0 {
			return nil, fmt.Errorf("proceeding with the order would put leverage ratio beyond its limit of %v to %v and cannot be placed", lookupRatio, ratio)
		}
		lookupRate := r.MaxLeverageRate[o.GetExchange()][o.GetAssetType()][o.Pair()]
		if retOrder.GetLeverage() > lookupRate && lookupRate > 0 {
			{
				return nil, fmt.Errorf("proceeding with the order would put leverage rate beyond its limit of %v to %v and cannot be placed", lookupRate, retOrder.GetLeverage())
			}
		}
	}
	if len(latestHoldings) > 1 {
		ratio := assessHoldingsRatio(o.Pair(), latestHoldings)
		lookupHolding := r.MaximumHoldingRatio[o.GetExchange()][o.GetAssetType()][o.Pair()]
		if lookupHolding > 0 && ratio > lookupHolding {
			return nil, fmt.Errorf("proceeding with the order would put holdings ratio beyond its limit of %v to %v and cannot be placed", lookupHolding, ratio)
		}
	}
	return retOrder, nil
}

func existingLeverageRatio(s compliance.Snapshot) float64 {
	if len(s.Orders) == 0 {
		return 0
	}
	var ordersWithLeverage float64
	for o := range s.Orders {
		if s.Orders[o].Leverage != "" {
			ordersWithLeverage++
		}
	}
	return ordersWithLeverage / float64(len(s.Orders))
}

// add additional assessing rules, such as what the maximum ratio is allowed to be
func assessHoldingsRatio(c currency.Pair, h []holdings.Holding) float64 {
	resp := make(map[currency.Pair]float64)
	for i := range h {
		resp[h[i].Pair] += h[i].PositionsSize
	}
	totalPosition := 0.0

	for _, v := range resp {
		totalPosition += v
	}
	ratio := resp[c] / totalPosition

	return ratio
}
