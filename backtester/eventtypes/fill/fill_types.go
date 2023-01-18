package fill

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Fill is an event that details the events from placing an order
type Fill struct {
	*event.Base
	Direction           order.Side      `json:"side"`
	Amount              decimal.Decimal `json:"amount"`
	ClosePrice          decimal.Decimal `json:"close-price"`
	VolumeAdjustedPrice decimal.Decimal `json:"volume-adjusted-price"`
	PurchasePrice       decimal.Decimal `json:"purchase-price"`
	Total               decimal.Decimal `json:"total"`
	ExchangeFee         decimal.Decimal `json:"exchange-fee"`
	Slippage            decimal.Decimal `json:"slippage"`
	Order               *order.Detail   `json:"-"`
	FillDependentEvent  signal.Event
	Liquidated          bool
}

// Event holds all functions required to handle a fill event
type Event interface {
	common.Event
	common.Directioner

	SetAmount(decimal.Decimal)
	GetAmount() decimal.Decimal
	GetClosePrice() decimal.Decimal
	GetVolumeAdjustedPrice() decimal.Decimal
	GetSlippageRate() decimal.Decimal
	GetPurchasePrice() decimal.Decimal
	GetTotal() decimal.Decimal
	GetExchangeFee() decimal.Decimal
	SetExchangeFee(decimal.Decimal)
	GetOrder() *order.Detail
	GetFillDependentEvent() signal.Event
	IsLiquidated() bool
}
