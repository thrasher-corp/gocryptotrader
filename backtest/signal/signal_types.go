package signal

import "github.com/thrasher-corp/gocryptotrader/backtest/event"

type Handler interface {
	event.Handler
	event.Direction
}
