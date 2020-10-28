package statistics

import (
	"time"

	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/fill"
	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/results"
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
	Pair         string
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
	ReturnResults() results.Results
	Reset()

	SetStrategyName(string)
}
