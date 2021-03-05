package fill

import (
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
func (f *Fill) SetAmount(i float64) {
	f.Amount = i
}

// GetAmount returns the amount
func (f *Fill) GetAmount() float64 {
	return f.Amount
}

// GetClosePrice returns the closing price
func (f *Fill) GetClosePrice() float64 {
	return f.ClosePrice
}

// GetVolumeAdjustedPrice returns the volume adjusted price
func (f *Fill) GetVolumeAdjustedPrice() float64 {
	return f.VolumeAdjustedPrice
}

// GetPurchasePrice returns the purchase price
func (f *Fill) GetPurchasePrice() float64 {
	return f.PurchasePrice
}

// GetTotal returns the total cost
func (f *Fill) GetTotal() float64 {
	return f.Total
}

// GetExchangeFee returns the exchange fee
func (f *Fill) GetExchangeFee() float64 {
	return f.ExchangeFee
}

// SetExchangeFee sets the exchange fee
func (f *Fill) SetExchangeFee(fee float64) {
	f.ExchangeFee = fee
}

// GetOrder returns the order
func (f *Fill) GetOrder() *order.Detail {
	return f.Order
}

// GetSlippageRate returns the slippage rate
func (f *Fill) GetSlippageRate() float64 {
	return f.Slippage
}
