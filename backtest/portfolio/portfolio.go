package portfolio

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/data"
	"github.com/thrasher-corp/gocryptotrader/backtest/event"
	"github.com/thrasher-corp/gocryptotrader/backtest/order"
	"github.com/thrasher-corp/gocryptotrader/backtest/position"
	"github.com/thrasher-corp/gocryptotrader/backtest/position/fill"
	"github.com/thrasher-corp/gocryptotrader/backtest/signal"
)

func (p Portfolio) Update(handler data.Handler) {

}

func (p Portfolio) Reset() {
	p.funds = 0
	p.holdings = nil
	p.transactions = nil
}

func (p Portfolio) Initial() float64 {
	return 0
}

func (p Portfolio) SetInitial(f float64) {
	p.initialFunds = f
}

func (p Portfolio) Funds() float64 {
	return p.funds
}

func (p Portfolio) SetFunds(f float64) {
	p.funds = f
}

func (p Portfolio) OnSignal(signal signal.Handler, data data.Handler) (*order.Order, error) {
	newOrder := &order.Order{
		Event:              event.Event{
			Timestamp: signal.Time(),
			Pair: signal.Pair(),
		},
		Direction:          signal.Direction(),
	}
	return newOrder, nil
}

func (p Portfolio) OnFill(fillH fill.Handler, data data.Handler) (*fill.Event, error) {
	if p.holdings == nil {
		p.holdings = make(map[string]position.Position)
	}

	if pos, ok := p.holdings[fillH.Pair().String()]; ok {
		pos.Update(fillH)
		p.holdings[fillH.Pair().String()] = pos
	} else {
		pos := position.Position{}
		pos.Create(fillH)
		p.holdings[fillH.Pair().String()] = pos
	}

	if fillH.Direction() == event.BUY {
		p.funds -= fillH.NetValue()
	} else {
		p.funds += fillH.NetValue()
	}
	p.transactions = append(p.transactions, fillH)
	return fillH.(*fill.Event), nil
}