package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type DataHandler interface {
	DataLoader
	DataStreamer
	Reset()
}

type DataLoader interface {
	Load() error
}

type DataStreamer interface {
	Next() (DataEventHandler, bool)
	Stream() []DataEventHandler
	History() []DataEventHandler
	Latest(pair currency.Pair) DataEventHandler
	List(pair currency.Pair) []DataEventHandler
}

type EventHandler interface {
	IsEvent() bool
	GetTime() time.Time
	Pair
}

type Pair interface {
	Pair() currency.Pair
}

type DataEventHandler interface {
	EventHandler
	LatestPrice() float64
}

type CandleEvent interface {
	DataEventHandler
	IsCandles() bool
}

type TickEvent interface {
	DataEventHandler
	IsTick() bool
}

type SignalEvent interface {
	EventHandler
	Directioner

	SetAmount(float64)
	GetAmount() float64
	GetPrice() float64
	IsSignal() bool
}

type OrderEvent interface {
	EventHandler
	Directioner

	SetAmount(float64)
	GetAmount() float64
	IsOrder() bool
}

type Directioner interface {
	SetDirection(side order.Side)
	GetDirection() order.Side
}

type FillEvent interface {
	EventHandler
	Directioner

	SetAmount(float64)
	GetAmount() float64
	GetPrice() float64
	GetCommission() float64
	GetExchangeFee() float64
	GetCost() float64
	Value() float64
	NetValue() float64
}

type ExecutionHandler interface {
	ExecuteOrder(OrderEvent, DataHandler) (*Fill, error)
}

type StatisticHandler interface {
	EventTracker
	Update(DataEventHandler, PortfolioHandler)
	TrackTransaction(FillEvent)
	Transactions() []FillEvent

	TotalEquityReturn() (float64, error)

	MaxDrawdown() float64
	MaxDrawdownTime() time.Time
	MaxDrawdownDuration() time.Duration

	SharpRatio(float64) float64
	SortinoRatio(float64) float64

	PrintResult()
	Reset()
}

type EventTracker interface {
	TrackEvent(EventHandler)
	Events() []EventHandler
}

type TransactionTracker interface {
	TrackTransaction(FillEvent)
	Transactions() []FillEvent
}

type PortfolioHandler interface {
	OnSignal(SignalEvent, DataHandler) (*Order, error)
	OnFill(FillEvent, DataHandler) (*Fill, error)
	Update(DataEventHandler)

	SetInitialFunds(float64)
	InitialFunds() float64
	SetFunds(float64)
	Funds() float64

	Value() float64
	ViewHoldings()

	Reset()
}

type StrategyHandler interface {
	OnSignal(DataEventHandler, DataHandler, PortfolioHandler) (SignalEvent, error)
}

type RiskHandler interface {
	EvaluateOrder(OrderEvent, DataEventHandler, map[string]Positions) (*Order, error)
}

type SizeHandler interface {
	SizeOrder(OrderEvent, DataEventHandler, PortfolioHandler) (*Order, error)
}
