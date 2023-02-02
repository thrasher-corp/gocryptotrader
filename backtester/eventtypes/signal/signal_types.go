package signal

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Event handler is used for getting trade signal details
// Example Amount and Price of current candle tick
type Event interface {
	common.Event
	common.Directioner
	ToKline() kline.Event
	GetClosePrice() decimal.Decimal
	GetHighPrice() decimal.Decimal
	GetOpenPrice() decimal.Decimal
	GetLowPrice() decimal.Decimal
	GetVolume() decimal.Decimal
	IsSignal() bool
	GetSellLimit() decimal.Decimal
	GetBuyLimit() decimal.Decimal
	GetAmount() decimal.Decimal
	GetFillDependentEvent() Event
	GetCollateralCurrency() currency.Code
	SetAmount(decimal.Decimal)
	MatchOrderAmount() bool
	IsNil() bool
}

// Signal contains everything needed for a strategy to raise a signal event
type Signal struct {
	*event.Base
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
	FillDependentEvent Event
	// CollateralCurrency is an optional parameter
	// when using futures to limit the collateral available
	// to a singular currency
	// eg with $5000 usd and 1 BTC, specifying BTC ensures
	// the USD value won't be utilised when sizing an order
	CollateralCurrency currency.Code
	// MatchOrderAmount flags to other event handlers
	// that the order amount must match the set Amount property
	MatchesOrderAmount bool
}
