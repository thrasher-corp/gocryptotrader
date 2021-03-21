package eventholder

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
)

// Reset returns struct to defaults
func (e *Holder) Reset() {
	e.Queue = nil
}

// AppendEvent adds and event to the queue
func (e *Holder) AppendEvent(i common.EventHandler) {
	e.Queue = append(e.Queue, i)
}

// NextEvent removes the current event and returns the next event in the queue
func (e *Holder) NextEvent() (i common.EventHandler) {
	if len(e.Queue) == 0 {
		return nil
	}

	i = e.Queue[0]
	e.Queue = e.Queue[1:]

	return i
}
