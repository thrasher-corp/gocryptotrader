package order

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// IsOrder returns whether the event is an order event
func (o *Order) IsOrder() bool {
	return true
}

// SetDirection sets the side of the order
func (o *Order) SetDirection(s order.Side) {
	o.Direction = s
}

// GetDirection returns the side of the order
func (o *Order) GetDirection() order.Side {
	return o.Direction
}

// SetAmount sets the amount
func (o *Order) SetAmount(i float64) {
	o.Amount = i
}

// GetAmount returns the amount
func (o *Order) GetAmount() float64 {
	return o.Amount
}

// GetBuyLimit returns the buy limit
func (o *Order) GetBuyLimit() float64 {
	return o.BuyLimit
}

// GetSellLimit returns the sell limit
func (o *Order) GetSellLimit() float64 {
	return o.SellLimit
}

// Pair returns the currency pair
func (o *Order) Pair() currency.Pair {
	return o.CurrencyPair
}

// GetStatus returns order status
func (o *Order) GetStatus() order.Status {
	return o.Status
}

// SetID sets the order id
func (o *Order) SetID(id string) {
	o.ID = id
}

// GetID returns the ID
func (o *Order) GetID() string {
	return o.ID
}

// IsLeveraged returns if it is leveraged
func (o *Order) IsLeveraged() bool {
	return o.Leverage > 1.0
}

// GetLeverage returns leverage rate
func (o *Order) GetLeverage() float64 {
	return o.Leverage
}

// SetLeverage sets leverage
func (o *Order) SetLeverage(l float64) {
	o.Leverage = l
}

// GetFunds returns the amount of funds the portfolio manager
// has allocated to this potential position
func (o *Order) GetFunds() float64 {
	return o.Funds
}
