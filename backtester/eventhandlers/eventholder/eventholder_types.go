package eventholder

import "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"

type Holder struct {
	Queue []interfaces.EventHandler
}

type EventHolder interface {
	Reset()
	AppendEvent(interfaces.EventHandler)
	NextEvent() (e interfaces.EventHandler, ok bool)
}
