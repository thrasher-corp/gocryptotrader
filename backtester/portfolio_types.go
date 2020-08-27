package backtest

import "github.com/thrasher-corp/gocryptotrader/currency"

type Portfolio struct {
	initialFunds float64
	funds        float64
	holdings     map[currency.Pair]Positions
	transactions []FillEvent
	sizeManager  SizeHandler
	riskManager  RiskHandler
}
