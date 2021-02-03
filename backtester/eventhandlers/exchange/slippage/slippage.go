package slippage

import (
	"math/rand"

	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// EstimateSlippagePercentage takes in an int range of numbers
// turns it into a percentage
func EstimateSlippagePercentage(maximumSlippageRate, minimumSlippageRate float64) float64 {
	if minimumSlippageRate < 1 || minimumSlippageRate > 100 {
		return 1
	}
	if maximumSlippageRate < 1 || maximumSlippageRate > 100 {
		return 1
	}

	// the language here is confusing. The maximum slippage rate is the lower bounds of the number,
	// eg 80 means for every dollar, keep 80%
	randSeed := int(minimumSlippageRate) - int(maximumSlippageRate)
	if randSeed > 0 {
		result := float64(rand.Intn(randSeed)) // nolint:gosec // basic number generation required, no need for crypto/rand
		return (result + maximumSlippageRate) / 100
	}
	return 1
}

// CalculateSlippageByOrderbook will analyse a provided orderbook and return the result of attempting to
// place the order on there
func CalculateSlippageByOrderbook(ob *orderbook.Base, side gctorder.Side, amountOfFunds, feeRate float64) (price, amount float64) {
	result := ob.SimulateOrder(amountOfFunds, side == gctorder.Buy)
	rate := (result.MinimumPrice - result.MaximumPrice) / result.MaximumPrice
	price = result.MinimumPrice * (rate + 1)
	amount = result.Amount * (1 - feeRate)
	return
}
