package slippage

import "github.com/thrasher-corp/gocryptotrader/types/decimal"

// Default slippage rates. It works on a percentage basis
// 100 means unaffected, 95 would mean 95%
var (
	DefaultMaximumSlippagePercent = decimal.NewFromInt(100)
	DefaultMinimumSlippagePercent = decimal.NewFromInt(100)
)
