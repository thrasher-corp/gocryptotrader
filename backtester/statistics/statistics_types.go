package statistics

import (
	"time"

	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Statistic
type Statistic struct {
	EventHistory       []portfolio.EventHandler
	TransactionHistory []fill.FillEvent
	Equity             []EquityPoint
	High               EquityPoint
	Low                EquityPoint
	InitialBuy         float64

	StrategyName string
}

type EquityPoint struct {
	Timestamp       time.Time
	Equity          float64
	EquityReturn    float64
	DrawnDown       float64
	BuyAndHoldValue float64
}

// StatisticHandler interface handles
type StatisticHandler interface {
	TrackEvent(portfolio.EventHandler)
	Events() []portfolio.EventHandler

	Update(portfolio.DataEventHandler, portfolio2.PortfolioHandler)
	TrackTransaction(fill.FillEvent)
	Transactions() []fill.FillEvent

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

type Results struct {
	Pair              string               `json:"pair"`
	TotalEvents       int                  `json:"totalEvents"`
	TotalTransactions int                  `json:"totalTransactions"`
	Events            []ResultEvent        `json:"events"`
	Transactions      []ResultTransactions `json:"transactions"`
	SharpieRatio      float64              `json:"sharpieRatio"`
	StrategyName      string               `json:"strategyName"`
}

type ResultTransactions struct {
	Time      time.Time  `json:"time"`
	Direction order.Side `json:"direction"`
	Price     float64    `json:"price"`
	Amount    float64    `json:"amount"`
	Why       string     `json:"why,omitempty"`
}

type ResultEvent struct {
	Time time.Time `json:"time"`
}
