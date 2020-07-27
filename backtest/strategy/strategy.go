package strategy

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/data"
	"github.com/thrasher-corp/gocryptotrader/backtest/portfolio"
)

func (s Strategy) Data() data.Handler {
	return nil
}

func (s Strategy) SetData(in data.Handler) {
	s.data = in
}

func (s Strategy) OnData(handler data.Handler) ([]Event, error) {
	return nil, nil
}

func (s Strategy) Portfolio() portfolio.Handler {
	return portfolio.Portfolio{}
}

func (s Strategy) SetPortfolio(in portfolio.Handler) {
}
