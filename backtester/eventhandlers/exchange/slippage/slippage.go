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
func CalculateSlippageByOrderbook(ob *orderbook.Base, side gctorder.Side, price float64) float64 {
	result, err := ob.WhaleBomb(price, side == gctorder.Buy)
	if err != nil {
		return 1
	}
	rate := (result.MinimumPrice - result.MaximumPrice) / result.MaximumPrice
	if rate < 0 {
		rate *= -1
	}
	if side == gctorder.Buy {
		return 1 + rate
	} else if side == gctorder.Sell {
		return 1 - rate
	}

	return 1
}
