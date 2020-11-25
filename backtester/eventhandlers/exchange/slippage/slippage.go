package slippage

import (
	"math/rand"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// EstimateSlippagePercentage takes in an int range of numbers
// turns it into a percentage
func EstimateSlippagePercentage(maximumSlippageRate, minimumSlippageRate float64, direction order.Side) float64 {
	if minimumSlippageRate < 0 || minimumSlippageRate > 100 {
		return 1
	}
	if maximumSlippageRate < 0 || maximumSlippageRate > 100 {
		return 1
	}
	if direction == order.Buy {

	} else if direction == order.Sell {
		// result := float64(rand.Intn(int(minimumSlippageRate) - int(maximumSlippageRate)))
	}
	// the language here is confusing. The maximum slippage rate is the lower bounds of the number,
	// eg 80 means for every dollar, keep 80%
	result := float64(rand.Intn(int(minimumSlippageRate) - int(maximumSlippageRate)))
	return (result + maximumSlippageRate) / 100
}

func CalculateSlippage(orderbook interface{}) float64 {
	return 1
}
