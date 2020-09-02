package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Statistic struct {
	eventHistory       []EventHandler
	transactionHistory []FillEvent
	equity             []EquityPoint
	high               EquityPoint
	low                EquityPoint
	initialBuy         float64

	strategyName string
}

type EquityPoint struct {
	timestamp       time.Time
	equity          float64
	equityReturn    float64
	drawnDown       float64
	buyAndHoldValue float64
}

type Results struct {
	TotalEvents       int                  `json:"totalEvents"`
	TotalTransactions int                  `json:"totalTransactions"`
	Transactions      []resultTransactions `json:"transactions"`
	SharpieRatio      float64              `json:"sharpieRatio"`
}

type resultTransactions struct {
	Time      time.Time  `json:"time"`
	Direction order.Side `json:"direction"`
	Price     float64    `json:"price"`
	Amount    float64    `json:"amount"`
}
