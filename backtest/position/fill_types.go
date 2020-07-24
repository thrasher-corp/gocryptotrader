package position

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/event"
	"github.com/thrasher-corp/gocryptotrader/backtest/signal"
)

type Fill struct {
	event.Handler

	Direction signal.Direction
	Amount float32
	Price float64
	Fee float64
	Total float64
}
