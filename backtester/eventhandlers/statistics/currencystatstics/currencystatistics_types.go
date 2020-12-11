package currencystatstics

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

type CurrencyStats interface {
	TotalEquityReturn() (float64, error)
	MaxDrawdown() Swing
	LongestDrawdown() Swing
	SharpeRatio(float64) float64
	SortinoRatio(float64) float64
}

type EventStore struct {
	Holdings      holdings.Holding
	Transactions  compliance.Snapshot
	DataEvent     interfaces.DataEventHandler
	SignalEvent   signal.SignalEvent
	ExchangeEvent order.OrderEvent
	FillEvent     fill.FillEvent
}

type CurrencyStatistic struct {
	Events                   []EventStore
	DrawDowns                SwingHolder
	Upswings                 SwingHolder
	LowestClosePrice         float64
	HighestClosePrice        float64
	MarketMovement           float64
	StrategyMovement         float64
	SharpeRatio              float64
	SortinoRatio             float64
	InformationRatio         float64
	RiskFreeRate             float64
	CalamariRatio            float64 // calmar
	CompoundAnnualGrowthRate float64
	BuyOrders                int64
	SellOrders               int64
}

// DrawdownHolder holds two types of drawdowns, the largest and longest
// it stores all of the calculated drawdowns
type SwingHolder struct {
	DrawDowns       []Swing
	MaxDrawDown     Swing
	LongestDrawDown Swing
}

// Swing holds a drawdown
type Swing struct {
	Highest            Iteration
	Lowest             Iteration
	CalculatedDrawDown float64
	Iterations         []Iteration
}

// Iteration is an individual iteration of price at a time
type Iteration struct {
	Time  time.Time
	Price float64
}
