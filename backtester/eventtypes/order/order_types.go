package order

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Order contains all details for an order event
type Order struct {
	event.Base
	ID        string
	Direction order.Side
	Status    order.Status
	Price     float64
	Amount    float64
	OrderType order.Type
	Leverage  float64
	Funds     float64
	BuyLimit  float64
	SellLimit float64
}

// Event inherits common event interfaces along with extra functions related to handling orders
type Event interface {
	common.EventHandler
	common.Directioner
	GetBuyLimit() float64
	GetSellLimit() float64
	SetAmount(float64)
	GetAmount() float64
	IsOrder() bool
	GetStatus() order.Status
	SetID(id string)
	GetID() string
	IsLeveraged() bool
	GetFunds() float64
}
