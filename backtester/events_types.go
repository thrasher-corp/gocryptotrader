package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Event struct {
	Time         time.Time
	CurrencyPair currency.Pair
}

type DataEvent struct {
	Metrics map[string]float64
}

type Signal struct {
	Event
	Amount    float64
	Price     float64
	Direction order.Side
}
