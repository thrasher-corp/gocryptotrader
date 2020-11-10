package ticker

import "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"

type Tick struct {
	event.Event
	Bid float64
	Ask float64
}
