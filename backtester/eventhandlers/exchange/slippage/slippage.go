package slippage

import (
	"math/rand/v2"

	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// EstimateSlippagePercentage takes in an int range of numbers
// turns it into a percentage
func EstimateSlippagePercentage(maximumSlippageRate, minimumSlippageRate decimal.Decimal) decimal.Decimal {
	if minimumSlippageRate.LessThan(decimal.NewFromInt(1)) || minimumSlippageRate.GreaterThan(decimal.NewFromInt(100)) {
		return decimal.NewFromInt(1)
	}
	if maximumSlippageRate.LessThan(decimal.NewFromInt(1)) || maximumSlippageRate.GreaterThan(decimal.NewFromInt(100)) {
		return decimal.NewFromInt(1)
	}

	// the language here is confusing. The maximum slippage rate is the lower bounds of the number,
	// eg 80 means for every dollar, keep 80%
	if randRange := minimumSlippageRate.IntPart() - maximumSlippageRate.IntPart(); randRange > 0 {
		result := rand.N(randRange) //nolint:gosec // basic number generation required, no need for crypto/rand
		return maximumSlippageRate.Add(decimal.NewFromInt(result)).Div(decimal.NewFromInt(100))
	}
	return decimal.NewFromInt(1)
}

// CalculateSlippageByOrderbook returns the price slippage for an order
func CalculateSlippageByOrderbook(ob *orderbook.Book, side gctorder.Side, allocatedFunds, feeRate decimal.Decimal) (price, amount decimal.Decimal, err error) {
	var result *orderbook.WhaleBombResult
	result, err = ob.SimulateOrder(allocatedFunds.InexactFloat64(), side == gctorder.Buy)
	if err != nil {
		return price, amount, err
	}
	rate := (result.MinimumPrice - result.MaximumPrice) / result.MaximumPrice
	price = decimal.MustFromFloat(result.MinimumPrice * (rate + 1))
	amount = decimal.MustFromFloat(result.Amount * (1 - feeRate.InexactFloat64()))
	return price, amount, err
}
