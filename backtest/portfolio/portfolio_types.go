package portfolio

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/data"
	"github.com/thrasher-corp/gocryptotrader/backtest/order"
	"github.com/thrasher-corp/gocryptotrader/backtest/position"
	"github.com/thrasher-corp/gocryptotrader/backtest/position/fill"
	"github.com/thrasher-corp/gocryptotrader/backtest/signal"
)

type Handler interface {
	Update(handler data.Handler)
	Reset()

	OnSignal(signal signal.Handler, data data.Handler) (*order.Order, error)
	OnFill(signal signal.Handler, data data.Handler) (*fill.Event, error)
	Funds
}

type Funds interface {
	Initial() float64
	SetInitial(float64)
	Funds() float64
	SetFunds(float64)
}

type Portfolio struct {
	initialFunds float64
	funds        float64

	holdings     map[string]position.Position
	transactions []fill.Handler
}
