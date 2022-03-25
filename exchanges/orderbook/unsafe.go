package orderbook

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
)

var errNoLiquidity = errors.New("no liquidity")

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
	*alert.Notice
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
func (d *Depth) GetUnsafe() *Unsafe {
	return &Unsafe{
		BidHead:        &d.bids.linkedList.head,
		AskHead:        &d.asks.linkedList.head,
		m:              &d.m,
		Notice:         &d.Notice,
		UpdatedViaREST: &d.options.restSnapshot,
		LastUpdated:    &d.options.lastUpdated,
	}
}

// CheckBidLiquidity determines if the liquidity is sufficient for usage
func (src *Unsafe) CheckBidLiquidity() error {
	_, err := src.GetBidLiquidity()
	return err
}

// CheckAskLiquidity determines if the liquidity is sufficient for usage
func (src *Unsafe) CheckAskLiquidity() error {
	_, err := src.GetAskLiquidity()
	return err
}

// GetBestBid returns the top bid price
func (src *Unsafe) GetBestBid() (float64, error) {
	bid, err := src.GetBidLiquidity()
	if err != nil {
		return 0, fmt.Errorf("get orderbook best bid price %w", err)
	}
	return bid.Value.Price, nil
}

// GetBestAsk returns the top ask price
func (src *Unsafe) GetBestAsk() (float64, error) {
	ask, err := src.GetAskLiquidity()
	if err != nil {
		return 0, fmt.Errorf("get orderbook best ask price %w", err)
	}
	return ask.Value.Price, nil
}

// GetBidLiquidity gets the head node for the bid liquidity
func (src *Unsafe) GetBidLiquidity() (*Node, error) {
	n := *src.BidHead
	if n == nil {
		return nil, fmt.Errorf("bid %w", errNoLiquidity)
	}
	return n, nil
}

// GetAskLiquidity gets the head node for the ask liquidity
func (src *Unsafe) GetAskLiquidity() (*Node, error) {
	n := *src.AskHead
	if n == nil {
		return nil, fmt.Errorf("ask %w", errNoLiquidity)
	}
	return n, nil
}

// GetLiquidity checks and returns nodes to the top bids and asks
func (src *Unsafe) GetLiquidity() (ask, bid *Node, err error) {
	bid, err = src.GetBidLiquidity()
	if err != nil {
		return nil, nil, err
	}
	ask, err = src.GetAskLiquidity()
	if err != nil {
		return nil, nil, err
	}
	return ask, bid, nil
}

// GetMidPrice returns the average between the top bid and top ask.
func (src *Unsafe) GetMidPrice() (float64, error) {
	ask, bid, err := src.GetLiquidity()
	if err != nil {
		return 0, fmt.Errorf("get orderbook mid price %w", err)
	}
	return (bid.Value.Price + ask.Value.Price) / 2, nil
}

// GetSpread returns the spread between the top bid and top asks.
func (src *Unsafe) GetSpread() (float64, error) {
	ask, bid, err := src.GetLiquidity()
	if err != nil {
		return 0, fmt.Errorf("get orderbook price spread %w", err)
	}
	return ask.Value.Price - bid.Value.Price, nil
}

// GetImbalance returns difference between the top bid and top ask amounts
// divided by its sum.
func (src *Unsafe) GetImbalance() (float64, error) {
	ask, bid, err := src.GetLiquidity()
	if err != nil {
		return 0, fmt.Errorf("get orderbook imbalance %w", err)
	}
	top := bid.Value.Amount - ask.Value.Amount
	bottom := bid.Value.Amount + ask.Value.Amount
	if bottom == 0 {
		return 0, errNoLiquidity
	}
	return top / bottom, nil
}

// IsStreaming returns if the orderbook is updated by a streaming protocol and
// is most likely more up to date than that of a REST protocol update.
func (src *Unsafe) IsStreaming() bool {
	return !*src.UpdatedViaREST
}
