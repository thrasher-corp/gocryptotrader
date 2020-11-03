package fill

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/direction"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Fill struct {
	event.Event
	Direction   order.Side
	Amount      float64
	Price       float64
	ExchangeFee float64
}

type FillEvent interface {
	interfaces.EventHandler
	direction.Directioner

	SetAmount(float64)
	GetAmount() float64
	GetPrice() float64
	GetExchangeFee() float64
	Value() float64
	NetValue() float64
}
