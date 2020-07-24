package backtest

import "github.com/thrasher-corp/gocryptotrader/backtest/event"

func New() *Backtest {
	return &Backtest{}
}

func (b *Backtest) Run() error {
	b.Portfolio.SetFunds(b.Portfolio.Initial())
	
	return nil
}

func (b *Backtest) event(handler event.Handler) error {
	return nil
}