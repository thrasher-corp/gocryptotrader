package ticker

import "github.com/thrasher-corp/gocryptotrader/backtester/event"

type Tick struct {
	event.Event
	Bid float64
	Ask float64
}
