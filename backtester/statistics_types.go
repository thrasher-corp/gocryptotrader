package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Statistic struct {
	eventHistory       []EventHandler
	transactionHistory []FillEvent
	equity             []equityPoint
	high               equityPoint
	low                equityPoint
	initialBuy         float64
}

type equityPoint struct {
	timestamp       time.Time
	equity          float64
	equityReturn    float64
	drawdown        float64
	buyAndHoldValue float64
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
