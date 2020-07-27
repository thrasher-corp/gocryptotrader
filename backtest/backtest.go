package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/data"
	"github.com/thrasher-corp/gocryptotrader/backtest/event"
	"github.com/thrasher-corp/gocryptotrader/backtest/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtest/strategy"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func New(pair currency.Pair, data data.Handler, pt portfolio.Handler, st strategy.Handler) *Backtest {
	return &Backtest{
		Pair:      pair,
		Data:      data,
		portfolio: pt,
		strategy:  st,
	}
}

func (b *Backtest) Run() error {
	b.portfolio.SetFunds(b.portfolio.Initial())
	b.strategy.SetData(b.Data)
	b.strategy.SetPortfolio(b.portfolio)

	for {
		e := b.event()
		if e == nil {
			d, ok := b.Data.Next()
			if !ok {
				break
			}
			b.eventQueue = append(b.eventQueue, d)
		}

		err := b.loop(e)
		if err != nil {
			return err
		}

	}

	return nil
}

func (b *Backtest) loop(in event.Handler) error {
	switch e := in.(type) {
	case data.Handler:
		b.portfolio.Update(e)

		signals, err := b.strategy.OnData(e)
		if err != nil {
			return err
		}
		for x := range signals {
			b.eventQueue = append(b.eventQueue, signals[x])
		}
	}
	return nil
}

func (b *Backtest) event() (out event.Handler) {
	if len(b.eventQueue) == 0 {
		return nil
	}
	out, b.eventQueue = b.eventQueue[0], b.eventQueue[1:]
	return out
}
