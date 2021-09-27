package signal

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Event handler is used for getting trade signal details
// Example Amount and Price of current candle tick
type Event interface {
	common.EventHandler
	common.Directioner

	GetPrice() decimal.Decimal
	IsSignal() bool
	GetSellLimit() decimal.Decimal
	GetBuyLimit() decimal.Decimal
}

// Signal contains everything needed for a strategy to raise a signal event
type Signal struct {
	event.Base
	OpenPrice  decimal.Decimal
	HighPrice  decimal.Decimal
	LowPrice   decimal.Decimal
	ClosePrice decimal.Decimal
	Volume     decimal.Decimal
	BuyLimit   decimal.Decimal
	SellLimit  decimal.Decimal
	Direction  order.Side
}
