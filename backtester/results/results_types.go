package results

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

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
}

type ResultEvent struct {
	Time time.Time `json:"time"`
}
