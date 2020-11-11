package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

// CandleEvent for OHLCV tick data
type CandleEvent interface {
	interfaces.DataEventHandler
}

// TickEvent interface for ticker data (bid/ask)
type TickEvent interface {
	interfaces.DataEventHandler
}
