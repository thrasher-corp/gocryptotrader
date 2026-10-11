//go:build !udecimal_on

package orderbook

import "github.com/thrasher-corp/gocryptotrader/types/decimal"

func scaleExecutionAmount(amount decimal.Decimal) decimal.Decimal {
	return amount
}

func unscaleExecutionAmount(amount decimal.Decimal) decimal.Decimal {
	return amount
}
