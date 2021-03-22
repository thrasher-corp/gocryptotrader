package fill

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Fill is an event that details the events from placing an order
type Fill struct {
	event.Base
	Direction           order.Side    `json:"side"`
	Amount              float64       `json:"amount"`
	ClosePrice          float64       `json:"close-price"`
	VolumeAdjustedPrice float64       `json:"volume-adjusted-price"`
	PurchasePrice       float64       `json:"purchase-price"`
	Total               float64       `json:"total"`
	ExchangeFee         float64       `json:"exchange-fee"`
	Slippage            float64       `json:"slippage"`
	Order               *order.Detail `json:"-"`
}

// Event holds all functions required to handle a fill event
type Event interface {
	common.EventHandler
	common.Directioner

	SetAmount(float64)
	GetAmount() float64
	GetClosePrice() float64
	GetVolumeAdjustedPrice() float64
	GetSlippageRate() float64
	GetPurchasePrice() float64
	GetTotal() float64
	GetExchangeFee() float64
	SetExchangeFee(float64)
	GetOrder() *order.Detail
}
