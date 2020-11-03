package backtest

import (
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

// CandleEvent for OHLCV tick data
type CandleEvent interface {
	portfolio.DataEventHandler
}

// TickEvent interface for ticker data (bid/ask)
type TickEvent interface {
	portfolio.DataEventHandler
}
