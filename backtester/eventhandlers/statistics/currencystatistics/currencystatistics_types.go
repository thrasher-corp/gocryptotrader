package currencystatistics

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
)

// CurrencyStats defines what is expected in order to
// calculate statistics based on an exchange, asset type and currency pair
type CurrencyStats interface {
	TotalEquityReturn() (float64, error)
	MaxDrawdown() Swing
	LongestDrawdown() Swing
	SharpeRatio(float64) float64
	SortinoRatio(float64) float64
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
	Events                   []EventStore          `json:"-"`
	MaxDrawdown              Swing                 `json:"max-drawdown,omitempty"`
	StartingClosePrice       float64               `json:"starting-close-price"`
	EndingClosePrice         float64               `json:"ending-close-price"`
	LowestClosePrice         float64               `json:"lowest-close-price"`
	HighestClosePrice        float64               `json:"highest-close-price"`
	MarketMovement           float64               `json:"market-movement"`
	StrategyMovement         float64               `json:"strategy-movement"`
	HighestCommittedFunds    HighestCommittedFunds `json:"highest-committed-funds"`
	RiskFreeRate             float64               `json:"risk-free-rate"`
	BuyOrders                int64                 `json:"buy-orders"`
	GeometricRatios          Ratios                `json:"geometric-ratios"`
	ArithmeticRatios         Ratios                `json:"arithmetic-ratios"`
	CompoundAnnualGrowthRate float64               `json:"compound-annual-growth-rate"`
	SellOrders               int64                 `json:"sell-orders"`
	TotalOrders              int64                 `json:"total-orders"`
	FinalHoldings            holdings.Holding      `json:"final-holdings"`
	FinalOrders              compliance.Snapshot   `json:"final-orders"`
	ShowMissingDataWarning   bool                  `json:"-"`
}

// Ratios stores all the ratios used for statistics
type Ratios struct {
	SharpeRatio      float64 `json:"sharpe-ratio"`
	SortinoRatio     float64 `json:"sortino-ratio"`
	InformationRatio float64 `json:"information-ratio"`
	CalmarRatio      float64 `json:"calmar-ratio"`
}

// Swing holds a drawdown
type Swing struct {
	Highest          Iteration `json:"highest"`
	Lowest           Iteration `json:"lowest"`
	DrawdownPercent  float64   `json:"drawdown"`
	IntervalDuration int64
}

// Iteration is an individual iteration of price at a time
type Iteration struct {
	Time  time.Time `json:"time"`
	Price float64   `json:"price"`
}

// HighestCommittedFunds is an individual iteration of price at a time
type HighestCommittedFunds struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}
