package portfolio

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"
	"github.com/thrasher-corp/gocryptotrader/backtester/positions"
	"github.com/thrasher-corp/gocryptotrader/backtester/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Portfolio struct {
	InitialFunds float64
	Funds        float64
	Holdings     map[currency.Pair]positions.Positions
	Transactions []fill.FillEvent
	SizeManager  SizeHandler
	RiskManager  risk.RiskHandler
}

type PortfolioHandler interface {
	OnSignal(signal.SignalEvent, datahandler.DataHandler) (*order.Order, error)
	OnFill(fill.FillEvent, datahandler.DataHandler) (*fill.Fill, error)
	Update(datahandler.DataEventHandler)

	SetInitialFunds(float64)
	GetInitialFunds() float64
	SetFunds(float64)
	GetFunds() float64

	Value() float64
	ViewHoldings() map[currency.Pair]positions.Positions

	Reset()
}

type SizeHandler interface {
	SizeOrder(orders.OrderEvent, datahandler.DataEventHandler) (*order.Order, error)
}
