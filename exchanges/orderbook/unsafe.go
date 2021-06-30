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
	m       *sync.Mutex
}

// Lock locks down the underlying linked list which inhibits all pending updates
// for strategy inspection.
func (src *Unsafe) Lock() {
	src.m.Lock()
}

// Unlock unlocks the underlying linked list after inspection by a strategy to
// resume normal operations
func (src *Unsafe) Unlock() {
	src.m.Unlock()
}

// Locker defines functionality for locking and unlocking unsafe books
type Locker interface {
	Lock()
	Unlock()
}

// LockWith locks both books for the context of cross orderbook inspection
func (src *Unsafe) LockWith(dst Locker) {
	src.m.Lock()
	dst.Lock()
}

// UnlockWith unlocks both books for the context of cross orderbook inspection
func (src *Unsafe) UnlockWith(dst Locker) {
	src.m.Unlock()
	dst.Unlock()
}

// GetUnsafe returns an unsafe orderbook with pointers to the linked list heads.
func (d *Depth) GetUnsafe() Unsafe {
	return Unsafe{
		BidHead: &d.bids.linkedList.head,
		AskHead: &d.asks.linkedList.head,
		m:       &d.m,
	}
}
