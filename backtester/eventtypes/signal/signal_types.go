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
	GetAmount() decimal.Decimal
}

// Signal contains everything needed for a strategy to raise a signal event
type Signal struct {
	event.Base
	OpenPrice  decimal.Decimal
	HighPrice  decimal.Decimal
	LowPrice   decimal.Decimal
	ClosePrice decimal.Decimal
	Volume     decimal.Decimal
	// BuyLimit sets a maximum buy from the strategy
	// it differs from amount as it is more a suggestion
	// use Amount if you wish to have a fillOrKill style amount
	BuyLimit decimal.Decimal
	// SellLimit sets a maximum sell from the strategy
	// it differs from amount as it is more a suggestion
	// use Amount if you wish to have a fillOrKill style amount
	SellLimit decimal.Decimal
	// Amount set the amount when you wish to allow
	// a strategy to dictate order quantities
	// if the amount is not allowed by the portfolio manager
	// the order will not be placed
	Amount    decimal.Decimal
	Direction order.Side
	// FillDependentEvent ensures that an order can only be placed
	// if there is corresponding collateral in the selected currency
	// this enabled cash and carry strategies for example
	FillDependentEvent *Signal
}
