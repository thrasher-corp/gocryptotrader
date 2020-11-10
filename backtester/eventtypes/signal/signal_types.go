package signal

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// SignalEvent handler is used for getting trade signal details
// Example Amount and Price of current candle tick
type SignalEvent interface {
	interfaces.EventHandler
	interfaces.Directioner

	SetAmount(float64)
	GetAmount() float64
	GetPrice() float64
	IsSignal() bool
	GetWhy() string
	SetWhy(string)
}

type Signal struct {
	event.Event
	Amount    float64
	Price     float64
	Direction order.Side
	Why       string
}
