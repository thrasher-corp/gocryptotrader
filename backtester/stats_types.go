package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const DP = 8

type StatisticHandler interface {
	Reset()

	TrackEvent(EventHandler)
	Events() []EventHandler

	TrackTransaction(OrderEvent)
	Transactions() []OrderEvent

	PrintResult()
	ReturnResult() Results

	Update(DataEvent, PortfolioHandler)

	TotalEquityReturn() (float64, error)
	MaxDrawdown() float64
	MaxDrawdownTime() time.Time
	MaxDrawdownDuration() time.Duration
	SharpRatio(float64) float64
	SortinoRatio(float64) float64
	GetEquity() *[]EquityPoint
}

type Statistic struct {
	eventHistory       []EventHandler
	transactionHistory []OrderEvent
	equity             []EquityPoint
	high               EquityPoint
	low                EquityPoint
}

type EquityPoint struct {
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
