package orderbook

import (
	"sync"
	"time"
)

// Unsafe is an exported linked list reference to the current bid/ask heads and
// a reference to the underlying depth mutex. This allows for the exposure of
// the internal list to an external strategy or subsystem. The bid and ask
// fields point to the actual head fields contained on both linked list structs,
// so that this struct can be reusable and not needed to be called on each
// inspection.
type Unsafe struct {
	BidHead **Node
	AskHead **Node
	m       *sync.Mutex

	// UpdatedViaREST defines if sync manager is updating this book via the REST
	// protocol then this book is not considered live and cannot be trusted.
	UpdatedViaREST *bool
	LastUpdated    *time.Time
	*Alert
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

// LockWith locks both books for the context of cross orderbook inspection.
// WARNING: When inspecting diametrically opposed books a higher order mutex
// MUST be used or a dead lock will occur.
func (src *Unsafe) LockWith(dst sync.Locker) {
	src.m.Lock()
	dst.Lock()
}

// UnlockWith unlocks both books for the context of cross orderbook inspection
func (src *Unsafe) UnlockWith(dst sync.Locker) {
	dst.Unlock() // Unlock in reverse order
	src.m.Unlock()
}

// GetUnsafe returns an unsafe orderbook with pointers to the linked list heads.
func (d *Depth) GetUnsafe() Unsafe {
	return Unsafe{
		BidHead:        &d.bids.linkedList.head,
		AskHead:        &d.asks.linkedList.head,
		m:              &d.m,
		Alert:          &d.Alert,
		UpdatedViaREST: &d.options.restSnapshot,
		LastUpdated:    &d.options.lastUpdated,
	}
}
