package fill

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// SetDirection sets the direction
func (f *Fill) SetDirection(s order.Side) {
	f.Direction = s
}

// GetDirection returns the direction
func (f *Fill) GetDirection() order.Side {
	return f.Direction
}

// SetAmount sets the amount
func (f *Fill) SetAmount(i decimal.Decimal) {
	f.Amount = i
}

// GetAmount returns the amount
func (f *Fill) GetAmount() decimal.Decimal {
	return f.Amount
}

// GetClosePrice returns the closing price
func (f *Fill) GetClosePrice() decimal.Decimal {
	return f.ClosePrice
}

// GetVolumeAdjustedPrice returns the volume adjusted price
func (f *Fill) GetVolumeAdjustedPrice() decimal.Decimal {
	return f.VolumeAdjustedPrice
}

// GetPurchasePrice returns the purchase price
func (f *Fill) GetPurchasePrice() decimal.Decimal {
	return f.PurchasePrice
}

// GetTotal returns the total cost
func (f *Fill) GetTotal() decimal.Decimal {
	return f.Total
}

// GetExchangeFee returns the exchange fee
func (f *Fill) GetExchangeFee() decimal.Decimal {
	return f.ExchangeFee
}

// SetExchangeFee sets the exchange fee
func (f *Fill) SetExchangeFee(fee decimal.Decimal) {
	f.ExchangeFee = fee
}

// GetOrder returns the order
func (f *Fill) GetOrder() *order.Detail {
	return f.Order
}

// GetSlippageRate returns the slippage rate
func (f *Fill) GetSlippageRate() decimal.Decimal {
	return f.Slippage
}

// GetFillDependentEvent returns the fill dependent event
// to raise after a prerequisite event has been completed
func (f *Fill) GetFillDependentEvent() signal.Event {
	return f.FillDependentEvent
}

// IsLiquidated highlights if the fill event
// was a result of liquidation
func (f *Fill) IsLiquidated() bool {
	return f.Liquidated
}
