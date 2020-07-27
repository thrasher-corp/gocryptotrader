package strategy

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/data"
	"github.com/thrasher-corp/gocryptotrader/backtest/event"
	"github.com/thrasher-corp/gocryptotrader/backtest/portfolio"
)

type Handler interface {
	Data() data.Handler
	SetData(in data.Handler)
	OnData(handler data.Handler) ([]Event, error)

	Portfolio() portfolio.Handler
	SetPortfolio(in portfolio.Handler)
}

type Event interface {
	event.Handler
	event.Direction
}

type Strategy struct {
	data data.Handler
}
