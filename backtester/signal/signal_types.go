package signal

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/direction"
)

// SignalEvent handler is used for getting trade signal details
// Example Amount and Price of current candle tick
type SignalEvent interface {
	datahandler.EventHandler
	direction.Directioner

	SetAmount(float64)
	GetAmount() float64
	GetPrice() float64
	IsSignal() bool
}
