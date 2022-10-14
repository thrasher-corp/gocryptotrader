package eventholder

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

// Reset returns struct to defaults
func (h *Holder) Reset() error {
	if h == nil {
		return gctcommon.ErrNilPointer
	}
	h.Queue = nil
	return nil
}

// AppendEvent adds and event to the queue
func (h *Holder) AppendEvent(i common.Event) {
	h.Queue = append(h.Queue, i)
}

// NextEvent removes the current event and returns the next event in the queue
func (h *Holder) NextEvent() (i common.Event) {
	if len(h.Queue) == 0 {
		return nil
	}

	i = h.Queue[0]
	h.Queue = h.Queue[1:]

	return i
}
