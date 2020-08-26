package backtest

import (
	"math"
)

func (e *Exchange) ExecuteOrder(order OrderEvent, data DataHandler) (*Fill, error) {
	latest := data.Latest(order.Pair())
	f := &Fill{
		Event:    Event{Time: order.GetTime(), CurrencyPair: order.Pair()},
		Exchange: e.Symbol,
		Amount:   order.GetAmount(),
		Price:    latest.LatestPrice(),
	}
	f.Direction = order.GetDirection()
	f.Commission = e.calculateCommission(f.Amount, f.Price)
	f.ExchangeFee = e.calculateExchangeFee()
	f.Cost = e.calculateCost(f.Commission, f.ExchangeFee)

	return f, nil
}

func (e *Exchange) calculateCommission(amount, price float64) float64 {
	var comRate = e.CommissionRate
	return math.Floor(amount*price*comRate*10000) / 10000
}

func (e *Exchange) calculateExchangeFee() float64 {
	return e.ExchangeFee
}

func (e *Exchange) calculateCost(commission, fee float64) float64 {
	return commission + fee
}
