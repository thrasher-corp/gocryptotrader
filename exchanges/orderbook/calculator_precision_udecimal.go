//go:build udecimal_on

package orderbook

import "github.com/thrasher-corp/gocryptotrader/types/decimal"

// udecimal truncates each product to 19 fractional digits. Scaling an amount
// by 10^19 before multiplying it by a price keeps each level's product exact,
// so later truncation only drops digits the unscaled totals would drop anyway.
var (
	executionScale   = decimal.MustFromString("10000000000000000000")
	executionUnscale = decimal.MustFromString("0.0000000000000000001")
)

func scaleExecutionAmount(amount decimal.Decimal) decimal.Decimal {
	return amount.Mul(executionScale)
}

func unscaleExecutionAmount(amount decimal.Decimal) decimal.Decimal {
	return amount.Mul(executionUnscale)
}
