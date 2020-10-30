package signal

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/direction"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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

type Signal struct {
	event.Event
	Amount    float64
	Price     float64
	Direction order.Side
}
