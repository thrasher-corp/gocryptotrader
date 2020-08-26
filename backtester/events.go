package backtest

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (e *Event) IsEvent() bool {
	return true
}

func (e *Event) GetTime() time.Time {
	return e.Time
}

func (e *Event) Pair() currency.Pair {
	return e.CurrencyPair
}

func (d *DataEvent) IsDataEvent() bool {
	return true
}

func (b *Candle) LatestPrice() float64 {
	return b.Close
}

func (t *Tick) LatestPrice() float64 {
	bid := decimal.NewFromFloat(t.Bid)
	ask := decimal.NewFromFloat(t.Ask)
	diff := decimal.New(2, 0)
	latest, _ := bid.Add(ask).Div(diff).Round(DP).Float64()
	return latest
}

func (s *Signal) IsSignal() bool {
	return true
}

func (s *Signal) SetDirection(st order.Side) {
	s.Direction = st
}

func (s *Signal) GetDirection() order.Side {
	return s.Direction
}

func (s *Signal) Pair() currency.Pair {
	return s.CurrencyPair
}

func (s *Signal) SetAmount(f float64) {
	s.Amount = f
}

func (s *Signal) GetAmount() float64 {
	return s.Amount
}

func (s *Signal) GetPrice() float64 {
	return s.Price
}

func (s *Signal) SetPrice(f float64) {
	s.Price = f
}
