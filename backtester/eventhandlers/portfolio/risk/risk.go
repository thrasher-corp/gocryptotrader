package risk

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// EvaluateOrder goes through a standard list of evaluations to make to ensure that
// we are in a position to follow through with an order
func (r *Risk) EvaluateOrder(o order.Event, latestHoldings []holdings.Holding, s compliance.Snapshot) (*order.Order, error) {
	if o == nil || latestHoldings == nil {
		return nil, gctcommon.ErrNilPointer
	}
	retOrder, ok := o.(*order.Order)
	if !ok {
		return nil, fmt.Errorf("%w expected order event", common.ErrInvalidDataType)
	}
	e := o.GetExchange()
	a := o.GetAssetType()
	p := o.Pair().Format(currency.EMPTYFORMAT)
	lookup, ok := r.CurrencySettings[key.NewExchangeAssetPair(e, a, p)]
	if !ok {
		return nil, fmt.Errorf("%v %v %v %w", e, a, p, errNoCurrencySettings)
	}

	if o.IsLeveraged() {
		if !r.CanUseLeverage {
			return nil, errLeverageNotAllowed
		}
		ratio := existingLeverageRatio(s)
		if ratio.GreaterThan(lookup.MaximumOrdersWithLeverageRatio) && lookup.MaximumOrdersWithLeverageRatio.GreaterThan(decimal.Zero) {
			return nil, fmt.Errorf("proceeding with the order would put maximum orders using leverage ratio beyond its limit of %v to %v and %w", lookup.MaximumOrdersWithLeverageRatio, ratio, errCannotPlaceLeverageOrder)
		}
		lr := lookup.MaxLeverageRate
		if retOrder.GetLeverage().GreaterThan(lr) && lr.GreaterThan(decimal.Zero) {
			return nil, fmt.Errorf("proceeding with the order would put leverage rate beyond its limit of %v to %v and %w", lookup.MaxLeverageRate, retOrder.GetLeverage(), errCannotPlaceLeverageOrder)
		}
	}
	if len(latestHoldings) > 1 {
		ratio := assessHoldingsRatio(o.Pair(), latestHoldings)
		if lookup.MaximumHoldingRatio.GreaterThan(decimal.Zero) && !ratio.Equal(decimal.NewFromInt(1)) && ratio.GreaterThan(lookup.MaximumHoldingRatio) {
			return nil, fmt.Errorf("order would exceed maximum holding ratio of %v to %v for %v %v %v. %w", lookup.MaximumHoldingRatio, ratio, e, a, p, errCannotPlaceLeverageOrder)
		}
	}
	return retOrder, nil
}

// existingLeverageRatio compares orders with leverage to the total number of orders
// a proof of concept to demonstrate risk manager's ability to prevent an order from being placed
// when an order exceeds a config setting
func existingLeverageRatio(s compliance.Snapshot) decimal.Decimal {
	if len(s.Orders) == 0 {
		return decimal.Zero
	}
	var ordersWithLeverage decimal.Decimal
	for o := range s.Orders {
		if s.Orders[o].Order.Leverage != 0 {
			ordersWithLeverage = ordersWithLeverage.Add(decimal.NewFromInt(1))
		}
	}
	return ordersWithLeverage.Div(decimal.NewFromInt(int64(len(s.Orders))))
}

func assessHoldingsRatio(c currency.Pair, h []holdings.Holding) decimal.Decimal {
	resp := make(map[currency.Pair]decimal.Decimal)
	totalPosition := decimal.Zero
	for i := range h {
		resp[h[i].Pair] = resp[h[i].Pair].Add(h[i].BaseValue)
		totalPosition = totalPosition.Add(h[i].BaseValue)
	}

	if totalPosition.IsZero() {
		return decimal.Zero
	}
	ratio := resp[c].Div(totalPosition)

	return ratio
}
