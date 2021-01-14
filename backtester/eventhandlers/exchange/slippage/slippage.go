package slippage

import (
	"math/rand"
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

func CalculateSlippage(orderbook interface{}) float64 {
	return 1
}
