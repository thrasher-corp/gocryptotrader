package size

import "github.com/thrasher-corp/gocryptotrader/backtester/config"

type Size struct {
	Leverage config.Leverage
	BuySide  config.MinMax
	SellSide config.MinMax
}
