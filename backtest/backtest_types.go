package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/event"
	"github.com/thrasher-corp/gocryptotrader/backtest/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtest/strategy"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Backtest struct {
	Pair currency.Pair

	Portfolio portfolio.Handler
	Strategy strategy.Handler
	Queue []event.Handler
}