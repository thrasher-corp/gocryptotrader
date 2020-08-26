package backtest

import (
	"github.com/shopspring/decimal"
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

func (f *Fill) GetPrice() float64 {
	return f.Price
}

func (f *Fill) GetCommission() float64 {
	return f.Commission
}

func (f *Fill) GetExchangeFee() float64 {
	return f.ExchangeFee
}

func (f *Fill) GetCost() float64 {
	return f.Cost
}

func (f *Fill) Value() float64 {
	amount := decimal.NewFromFloat(f.Amount)
	price := decimal.NewFromFloat(f.Price)
	value, _ := amount.Mul(price).Round(DP).Float64()
	return value
}

func (f *Fill) NetValue() float64 {
	amount := decimal.NewFromFloat(f.Amount)
	price := decimal.NewFromFloat(f.Price)
	cost := decimal.NewFromFloat(f.Cost)

	if f.Direction == order.Buy {
		netValue, _ := amount.Mul(price).Add(cost).Round(DP).Float64()
		return netValue
	}
	netValue, _ := amount.Mul(price).Sub(cost).Round(DP).Float64()
	return netValue
}
