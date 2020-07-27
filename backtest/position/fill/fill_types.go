package fill

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/event"
)

type Event struct {
	event.Handler
	event.Direction

	Amount float64
	Price  float64
	Fee    float64
}
