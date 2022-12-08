package order

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
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
func (o *Order) SetAmount(i decimal.Decimal) {
	o.Amount = i
}

// GetAmount returns the amount
func (o *Order) GetAmount() decimal.Decimal {
	return o.Amount
}

// GetBuyLimit returns the buy limit
func (o *Order) GetBuyLimit() decimal.Decimal {
	return o.BuyLimit
}

// GetSellLimit returns the sell limit
func (o *Order) GetSellLimit() decimal.Decimal {
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
	return o.Leverage.GreaterThan(decimal.NewFromFloat(1))
}

// GetLeverage returns leverage rate
func (o *Order) GetLeverage() decimal.Decimal {
	return o.Leverage
}

// SetLeverage sets leverage
func (o *Order) SetLeverage(l decimal.Decimal) {
	o.Leverage = l
}

// GetAllocatedFunds returns the amount of funds the portfolio manager
// has allocated to this potential position
func (o *Order) GetAllocatedFunds() decimal.Decimal {
	return o.AllocatedFunds
}

// GetFillDependentEvent returns the fill dependent event
// so it can be added the event queue
func (o *Order) GetFillDependentEvent() signal.Event {
	return o.FillDependentEvent
}

// IsClosingPosition returns whether position is being closed
func (o *Order) IsClosingPosition() bool {
	return o.ClosingPosition
}

// IsLiquidating returns whether position is being liquidated
func (o *Order) IsLiquidating() bool {
	return o.LiquidatingPosition
}

// GetClosePrice returns the close price
func (o *Order) GetClosePrice() decimal.Decimal {
	return o.ClosePrice
}
