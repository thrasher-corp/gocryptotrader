package event

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Event struct {
	Time         time.Time
	CurrencyPair currency.Pair
}

type Signal struct {
	Event
	Amount    float64
	Price     float64
	Direction order.Side
}
