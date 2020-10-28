package exchange

import (
	"math"

	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/orderbook"
)

func (e *Exchange) ExecuteOrder(o orderbook.OrderEvent, data portfolio.DataHandler) (*fill.Fill, error) {
	f := &fill.Fill{
		Event: event.Event{
			Time:         o.GetTime(),
			CurrencyPair: o.Pair(),
		},
		Amount: o.GetAmount(),
		Price:  data.Latest().LatestPrice(),
	}
	f.Amount = 1
	f.Direction = o.GetDirection()
	f.Commission = e.calculateCommission(f.Amount, f.Price)
	f.ExchangeFee = e.calculateExchangeFee()
	f.Cost = e.calculateCost(f.Commission, f.ExchangeFee)

	return f, nil
}

func (e *Exchange) calculateCommission(amount, price float64) float64 {
	return math.Floor(amount*price*e.CommissionRate*10000) / 10000
}

func (e *Exchange) calculateExchangeFee() float64 {
	return e.ExchangeFee
}

func (e *Exchange) calculateCost(commission, fee float64) float64 {
	return commission + fee
}
