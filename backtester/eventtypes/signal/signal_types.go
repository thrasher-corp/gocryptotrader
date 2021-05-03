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

	GetPrice() float64
	IsSignal() bool
	GetSellLimit() float64
	GetBuyLimit() float64
}

// Signal contains everything needed for a strategy to raise a signal event
type Signal struct {
	event.Base
	OpenPrice  float64
	HighPrice  float64
	LowPrice   float64
	ClosePrice float64
	Volume     float64
	BuyLimit   float64
	SellLimit  float64
	Direction  order.Side
}
