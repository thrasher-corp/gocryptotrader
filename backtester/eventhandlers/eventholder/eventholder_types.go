package eventholder

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
)

// Holder contains the event queue for backtester processing
type Holder struct {
	Queue           []common.EventHandler
	RunTimer        time.Duration
	NewEventTimeout time.Duration
	DataCheckTimer  time.Duration
}

// EventHolder interface details what is expected of an event holder to perform
type EventHolder interface {
	Reset()
	AppendEvent(common.EventHandler)
	NextEvent() common.EventHandler
	GetRunTimer() time.Duration
	GetNewEventTimeout() time.Duration
	GetDataCheckTimer() time.Duration
}
