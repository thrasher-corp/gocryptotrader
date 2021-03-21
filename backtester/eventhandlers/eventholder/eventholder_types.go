package eventholder

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
)

// Holder contains the event queue for backtester processing
type Holder struct {
	Queue []common.EventHandler
}

// EventHolder interface details what is expected of an event holder to perform
type EventHolder interface {
	Reset()
	AppendEvent(common.EventHandler)
	NextEvent() (e common.EventHandler)
}
