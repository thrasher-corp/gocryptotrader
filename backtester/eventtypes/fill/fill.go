package fill

import (
	"github.com/shopspring/decimal"
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

// GetTradingFee returns the exchange trading fee
func (f *Fill) GetTradingFee() decimal.Decimal {
	return f.TradingFee
}

// SetTradingFee sets the exchange trading fee
func (f *Fill) SetTradingFee(fee decimal.Decimal) {
	f.TradingFee = fee
}

// GetOrder returns the order
func (f *Fill) GetOrder() *order.Detail {
	return f.Order
}

// GetSlippageRate returns the slippage rate
func (f *Fill) GetSlippageRate() decimal.Decimal {
	return f.Slippage
}
