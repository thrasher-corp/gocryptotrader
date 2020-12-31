package eventholder

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
)

type Holder struct {
	Queue []common.EventHandler
}

type EventHolder interface {
	Reset()
	AppendEvent(common.EventHandler)
	NextEvent() (e common.EventHandler, ok bool)
}
