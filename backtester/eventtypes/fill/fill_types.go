package fill

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Fill struct {
	event.Event
	Direction           order.Side    `json:"side"`
	Amount              float64       `json:"amount"`
	ClosePrice          float64       `json:"close-price"`
	VolumeAdjustedPrice float64       `json:"volume-adjusted-price"`
	PurchasePrice       float64       `json:"purchase-price"`
	ExchangeFee         float64       `json:"exchange-fee"`
	Slippage            float64       `json:"slippage"`
	Order               *order.Detail `json:"-"`
}

type Event interface {
	common.EventHandler
	common.Directioner

	SetAmount(float64)
	GetAmount() float64
	GetClosePrice() float64
	GetVolumeAdjustedPrice() float64
	GetSlippageRate() float64
	GetPurchasePrice() float64
	GetExchangeFee() float64
	SetExchangeFee(float64)
	Value() float64
	NetValue() float64
	GetOrder() *order.Detail
}
