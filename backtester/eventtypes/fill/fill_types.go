package fill

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Fill struct {
	event.Event
	Direction           order.Side
	Amount              float64
	ClosePrice          float64
	VolumeAdjustedPrice float64
	PurchasePrice       float64
	ExchangeFee         float64
	Slippage            float64
	Order               *order.Detail
}

type FillEvent interface {
	interfaces.EventHandler
	interfaces.Directioner

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
