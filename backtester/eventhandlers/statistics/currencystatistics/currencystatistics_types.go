package currencystatistics

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
)

var (
	roundTo2 int32 = 2
	roundTo8 int32 = 8
)

// CurrencyStats defines what is expected in order to
// calculate statistics based on an exchange, asset type and currency pair
type CurrencyStats interface {
	TotalEquityReturn() (decimal.Decimal, error)
	MaxDrawdown() Swing
	LongestDrawdown() Swing
	SharpeRatio(decimal.Decimal) decimal.Decimal
	SortinoRatio(decimal.Decimal) decimal.Decimal
}

// EventStore is used to hold all event information
// at a time interval
type EventStore struct {
	Holdings     holdings.Holding
	Transactions compliance.Snapshot
	DataEvent    common.DataEventHandler
	SignalEvent  signal.Event
	OrderEvent   order.Event
	FillEvent    fill.Event
}

// CurrencyStatistic Holds all events and statistics relevant to an exchange, asset type and currency pair
type CurrencyStatistic struct {
	Events                       []EventStore          `json:"-"`
	MaxDrawdown                  Swing                 `json:"max-drawdown,omitempty"`
	StartingClosePrice           decimal.Decimal       `json:"starting-close-price"`
	EndingClosePrice             decimal.Decimal       `json:"ending-close-price"`
	LowestClosePrice             decimal.Decimal       `json:"lowest-close-price"`
	HighestClosePrice            decimal.Decimal       `json:"highest-close-price"`
	MarketMovement               decimal.Decimal       `json:"market-movement"`
	StrategyMovement             decimal.Decimal       `json:"strategy-movement"`
	HighestCommittedFunds        HighestCommittedFunds `json:"highest-committed-funds"`
	RiskFreeRate                 decimal.Decimal       `json:"risk-free-rate"`
	BuyOrders                    int64                 `json:"buy-orders"`
	GeometricRatios              Ratios                `json:"geometric-ratios"`
	ArithmeticRatios             Ratios                `json:"arithmetic-ratios"`
	CompoundAnnualGrowthRate     decimal.Decimal       `json:"compound-annual-growth-rate"`
	SellOrders                   int64                 `json:"sell-orders"`
	TotalOrders                  int64                 `json:"total-orders"`
	InitialHoldings              holdings.Holding      `json:"initial-holdings-holdings"`
	FinalHoldings                holdings.Holding      `json:"final-holdings"`
	FinalOrders                  compliance.Snapshot   `json:"final-orders"`
	ShowMissingDataWarning       bool                  `json:"-"`
	IsStrategyProfitable         bool                  `json:"is-strategy-profitable"`
	DoesPerformanceBeatTheMarket bool                  `json:"does-performance-beat-the-market"`
}

// Ratios stores all the ratios used for statistics
type Ratios struct {
	SharpeRatio      decimal.Decimal `json:"sharpe-ratio"`
	SortinoRatio     decimal.Decimal `json:"sortino-ratio"`
	InformationRatio decimal.Decimal `json:"information-ratio"`
	CalmarRatio      decimal.Decimal `json:"calmar-ratio"`
}

// Swing holds a drawdown
type Swing struct {
	Highest          Iteration       `json:"highest"`
	Lowest           Iteration       `json:"lowest"`
	DrawdownPercent  decimal.Decimal `json:"drawdown"`
	IntervalDuration int64
}

// Iteration is an individual iteration of price at a time
type Iteration struct {
	Time  time.Time       `json:"time"`
	Price decimal.Decimal `json:"price"`
}

// HighestCommittedFunds is an individual iteration of price at a time
type HighestCommittedFunds struct {
	Time  time.Time       `json:"time"`
	Value decimal.Decimal `json:"value"`
}
