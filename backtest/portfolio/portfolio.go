package portfolio

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/data"
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
	return 0
}

func (p Portfolio) SetFunds(f float64) {
	p.funds = f
}


func (p Portfolio) OnSignal(signal signal.Handler, data data.Handler) error {
	return nil
}

