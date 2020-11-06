package portfolio

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
	"github.com/thrasher-corp/gocryptotrader/backtester/risk"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Portfolio struct {
	InitialFunds           float64
	Funds                  float64
	FundsPerCurrency       map[currency.Pair]float64
	Holdings               map[currency.Pair]positions.Positions
	Transactions           []fill.FillEvent
	SizeManager            SizeHandler
	SizeManagerPerCurrency map[currency.Pair]SizeHandler
	RiskManager            risk.RiskHandler
}

type PortfolioHandler interface {
	OnSignal(signal.SignalEvent, interfaces.DataHandler) (*order.Order, error)
	OnFill(fill.FillEvent, interfaces.DataHandler) (*fill.Fill, error)
	Update(interfaces.DataEventHandler)

	SetInitialFunds(float64)
	GetInitialFunds() float64
	SetFunds(currency.Pair, float64)
	GetFunds(currency.Pair) float64

	Value() float64
	ViewHoldings() map[currency.Pair]positions.Positions

	Reset()
}

type SizeHandler interface {
	SizeOrder(orders.OrderEvent, interfaces.DataEventHandler, float64, float64) (*order.Order, error)
}
