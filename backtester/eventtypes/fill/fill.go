package fill

import (
	"github.com/shopspring/decimal"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (f *Fill) SetDirection(s order.Side) {
	f.Direction = s
}

func (f *Fill) GetDirection() order.Side {
	return f.Direction
}

func (f *Fill) SetAmount(i float64) {
	f.Amount = i
}

func (f *Fill) GetAmount() float64 {
	return f.Amount
}

func (f *Fill) GetWhy() string {
	return f.Why
}

func (f *Fill) GetClosePrice() float64 {
	return f.ClosePrice
}

func (f *Fill) GetVolumeAdjustedPrice() float64 {
	return f.VolumeAdjustedPrice
}

func (f *Fill) GetPurchasePrice() float64 {
	return f.PurchasePrice
}

func (f *Fill) GetExchangeFee() float64 {
	return f.ExchangeFee
}

func (f *Fill) SetExchangeFee(fee float64) {
	f.ExchangeFee = fee
}

func (f *Fill) Value() float64 {
	amount := decimal.NewFromFloat(f.Amount)
	price := decimal.NewFromFloat(f.PurchasePrice)
	value, _ := amount.Mul(price).Round(common.DecimalPlaces).Float64()
	return value
}

func (f *Fill) NetValue() float64 {
	amount := decimal.NewFromFloat(f.Amount)
	price := decimal.NewFromFloat(f.PurchasePrice)
	fee := decimal.NewFromFloat(f.ExchangeFee)
	if f.Direction == order.Buy {
		netValue, _ := amount.Mul(price).Add(fee).Round(common.DecimalPlaces).Float64()
		return netValue
	}
	netValue, _ := amount.Mul(price).Sub(fee).Round(common.DecimalPlaces).Float64()
	return netValue
}

func (f *Fill) GetOrder() *order.Detail {
	return f.Order
}
