package eventholder

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
)

func (e *Holder) Reset() {
	e.Queue = nil
}

func (e *Holder) AppendEvent(i common.EventHandler) {
	e.Queue = append(e.Queue, i)
}

func (e *Holder) NextEvent() (i common.EventHandler, ok bool) {
	if len(e.Queue) == 0 {
		return i, false
	}

	i = e.Queue[0]
	e.Queue = e.Queue[1:]

	return i, true
}
