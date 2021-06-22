package orderbook

import "sync"

// Unsafe is an exported linked list reference to current the current bid/ask
// heads and a reference to the underlying depth mutex. This allows for the
// exposure of the internal list to a strategy or subsystem. The bid and ask
// fields point to the actual in struct head field so that this struct can be
// reusable and not needed to be called on each inspection.
type Unsafe struct {
	BidHead **Node
	AskHead **Node
	M       *sync.Mutex
}

// GetUnsafe returns an unsafe orderbook with pointers to the linked list heads.
func (d *Depth) GetUnsafe() Unsafe {
	return Unsafe{
		BidHead: &d.bids.linkedList.head,
		AskHead: &d.asks.linkedList.head,
		M:       &d.m,
	}
}
