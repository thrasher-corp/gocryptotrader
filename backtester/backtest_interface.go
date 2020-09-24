package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// DataHandler interface for Loading and Streaming data
type DataHandler interface {
	DataLoader
	DataStreamer
	Reset()
}

// DataLoader interface for Loading data into backtest supported format
type DataLoader interface {
	Load() error
}

// DataStreamer interface handles loading, parsing, distributing BackTest data
type DataStreamer interface {
	Next() (DataEventHandler, bool)
	Stream() []DataEventHandler
	History() []DataEventHandler
	Latest() DataEventHandler
	List() []DataEventHandler
	Offset() int

	StreamOpen() []float64
	StreamHigh() []float64
	StreamLow() []float64
	StreamClose() []float64
	StreamVol() []float64
}

// EventHandler interface implements required GetTime() & Pair() return
type EventHandler interface {
	IsEvent() bool
	GetTime() time.Time
	Pair() currency.Pair
}

// DataHandler interface used for loading and interacting with Data
type DataEventHandler interface {
	EventHandler
	DataType() DataType
	LatestPrice() float64
}

// CandleEvent for OHLCV tick data
type CandleEvent interface {
	DataEventHandler
}

// TickEvent interface for ticker data (bid/ask)
type TickEvent interface {
	DataEventHandler
}

// SignalEvent handler is used for getting trade signal details
// Example Amount and Price of current candle tick
type SignalEvent interface {
	EventHandler
	Directioner

	SetAmount(float64)
	GetAmount() float64
	GetPrice() float64
	IsSignal() bool
}

// OrderEvent
type OrderEvent interface {
	EventHandler
	Directioner

	SetAmount(float64)
	GetAmount() float64
	IsOrder() bool

	GetStatus() order.Status
	SetID(id int)
	ID() int
	Limit() float64
	IsLeveraged() bool
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

// StatisticHandler interface handles
type StatisticHandler interface {
	TrackEvent(EventHandler)
	Events() []EventHandler

	Update(DataEventHandler, PortfolioHandler)
	TrackTransaction(FillEvent)
	Transactions() []FillEvent

	TotalEquityReturn() (float64, error)

	MaxDrawdown() float64
	MaxDrawdownTime() time.Time
	MaxDrawdownDuration() time.Duration

	SharpeRatio(float64) float64
	SortinoRatio(float64) float64

	PrintResult()
	ReturnResults() Results
	Reset()

	SetStrategyName(string)
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
	ViewHoldings() map[currency.Pair]Positions

	Reset()
}

type StrategyHandler interface {
	Name() string
	OnSignal(DataHandler, PortfolioHandler) (SignalEvent, error)
}

type RiskHandler interface {
	EvaluateOrder(OrderEvent, DataEventHandler, map[currency.Pair]Positions) (*Order, error)
}

type SizeHandler interface {
	SizeOrder(OrderEvent, DataEventHandler, PortfolioHandler) (*Order, error)
}
