package currencystatstics

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type CurrencyStats interface {
	TotalEquityReturn() (float64, error)
	MaxDrawdown() Swing
	LongestDrawdown() Swing
	SharpeRatio(float64) float64
	SortinoRatio(float64) float64
}

type EventStore struct {
	Holdings     holdings.Holding
	Transactions compliance.Snapshot
	DataEvent    common.DataEventHandler
	SignalEvent  signal.SignalEvent
	OrderEvent   order.OrderEvent
	FillEvent    fill.FillEvent
}

type CurrencyStatistic struct {
	Pair                     currency.Pair       `json:"pair"`
	Asset                    asset.Item          `json:"asset"`
	Exchange                 string              `json:"exchange"`
	Events                   []EventStore        `json:"-"`
	DrawDowns                SwingHolder         `json:"all-drawdowns,omitempty"`
	Upswings                 SwingHolder         `jons:"all-upswings,omitempty"`
	StartingClosePrice       float64             `json:"starting-close-price"`
	EndingClosePrice         float64             `json:"ending-close-price"`
	LowestClosePrice         float64             `json:"lowest-close-price"`
	HighestClosePrice        float64             `json:"highest-close-price"`
	MarketMovement           float64             `json:"market-movement"`
	StrategyMovement         float64             `json:"strategy-movement"`
	SharpeRatio              float64             `json:"sharpe-ratio"`
	SortinoRatio             float64             `json:"sortino-ratio"`
	InformationRatio         float64             `json:"information-ratio"`
	RiskFreeRate             float64             `json:"risk-free-rate"`
	CalamariRatio            float64             `json:"calmar-ratio"` // calmar
	CompoundAnnualGrowthRate float64             `json:"compound-annual-growth-rate"`
	BuyOrders                int64               `json:"buy-orders"`
	SellOrders               int64               `json:"sell-orders"`
	TotalOrders              int64               `json:"total-orders"`
	FinalHoldings            holdings.Holding    `json:"final-holdings"`
	FinalOrders              compliance.Snapshot `json:"final-orders"`
}

// DrawdownHolder holds two types of drawdowns, the largest and longest
// it stores all of the calculated drawdowns
type SwingHolder struct {
	DrawDowns       []Swing `json:"-"`
	MaxDrawDown     Swing   `json:"max-drawdown,omitempty"`
	LongestDrawDown Swing   `json:"longest-drawdown,omitempty"`
}

// Swing holds a drawdown
type Swing struct {
	Highest            Iteration   `json:"highest"`
	Lowest             Iteration   `json:"lowest"`
	CalculatedDrawDown float64     `json:"drawdown"`
	Iterations         []Iteration `json:"-"`
}

// Iteration is an individual iteration of price at a time
type Iteration struct {
	Time  time.Time `json:"time"`
	Price float64   `json:"price"`
}
