package signal

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Event handler is used for getting trade signal details
// Example Amount and Price of current candle tick
type Event interface {
	common.EventHandler
	common.Directioner

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
