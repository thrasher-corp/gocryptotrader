package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const DP = 0.000000000

type StatisticHandler interface {
	Reset()

	TrackEvent(EventHandler)
	Events() []EventHandler

	TrackTransaction(OrderEvent)
	Transactions() []OrderEvent

	PrintResult()

	Update(DataEvent, PortfolioHandler)

	TotalEquityReturn() (float64, error)
	MaxDrawdown() float64
	MaxDrawdownTime() time.Time
	MaxDrawdownDuration() time.Duration
	SharpRatio(float64) float64
	SortinoRatio(float64) float64
	GetEquity() *[]equityPoint
}

type Statistic struct {
	eventHistory       []EventHandler
	transactionHistory []OrderEvent
	equity             []equityPoint
	high               equityPoint
	low                equityPoint
}

type equityPoint struct {
	timestamp    time.Time
	equity       float64
	equityReturn float64
	drawdown     float64
}

type Results struct {
	TotalEvents       int
	TotalTransactions int
	Transactions      []resultTransactions
	SharpieRatio      float64
}

type resultTransactions struct {
	time      time.Time
	direction order.Side
	price     float64
	amount    float64
}
