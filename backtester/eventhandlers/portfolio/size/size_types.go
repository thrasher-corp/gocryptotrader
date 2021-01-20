package size

import "github.com/thrasher-corp/gocryptotrader/backtester/config"

// Size contains buy and sell side rules
type Size struct {
	BuySide  config.MinMax
	SellSide config.MinMax
}
