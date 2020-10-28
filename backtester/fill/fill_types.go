package fill

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/direction"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Fill struct {
	event.Event
	// Exchange    string
	Direction   order.Side
	Amount      float64
	Price       float64
	Commission  float64
	ExchangeFee float64
	Cost        float64
}

type FillEvent interface {
	datahandler.EventHandler
	direction.Directioner

	SetAmount(float64)
	GetAmount() float64
	GetPrice() float64
	GetCommission() float64
	GetExchangeFee() float64
	GetCost() float64
	Value() float64
	NetValue() float64
}
