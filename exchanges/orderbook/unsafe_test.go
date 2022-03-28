package orderbook

import (
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

var unsafeID, _ = uuid.NewV4()

type externalBook struct{}

func (e *externalBook) Lock()   {}
func (e *externalBook) Unlock() {}

func TestUnsafe(t *testing.T) {
	t.Parallel()
	d := NewDepth(unsafeID)
	ob := d.GetUnsafe()
	if ob.AskHead == nil || ob.BidHead == nil || ob.m == nil {
		t.Fatal("these items should not be nil")
	}

	ob2 := &externalBook{}
	ob.Lock()
	ob.Unlock() // nolint:staticcheck, gocritic // Not needed in test
	ob.LockWith(ob2)
	ob.UnlockWith(ob2)
}

func TestGetLiquidity(t *testing.T) {
	t.Parallel()
	d := NewDepth(unsafeID)
	unsafe := d.GetUnsafe()
	_, _, err := unsafe.GetLiquidity()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	d.LoadSnapshot([]Item{{Price: 2}}, nil, 0, time.Time{}, false)
	_, _, err = unsafe.GetLiquidity()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	d.LoadSnapshot([]Item{{Price: 2}}, []Item{{Price: 2}}, 0, time.Time{}, false)
	aN, bN, err := unsafe.GetLiquidity()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if aN == nil {
		t.Fatal("unexpected value")
	}

	if bN == nil {
		t.Fatal("unexpected value")
	}
}

func TestCheckBidLiquidity(t *testing.T) {
	t.Parallel()
	d := NewDepth(unsafeID)
	unsafe := d.GetUnsafe()
	err := unsafe.CheckBidLiquidity()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	d.LoadSnapshot([]Item{{Price: 2}}, nil, 0, time.Time{}, false)
	err = unsafe.CheckBidLiquidity()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestCheckAskLiquidity(t *testing.T) {
	t.Parallel()
	d := NewDepth(unsafeID)
	unsafe := d.GetUnsafe()
	err := unsafe.CheckAskLiquidity()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	d.LoadSnapshot(nil, []Item{{Price: 2}}, 0, time.Time{}, false)
	err = unsafe.CheckAskLiquidity()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetBestBid(t *testing.T) {
	t.Parallel()
	d := NewDepth(unsafeID)
	unsafe := d.GetUnsafe()
	if _, err := unsafe.GetBestBid(); !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	d.LoadSnapshot([]Item{{Price: 2}}, nil, 0, time.Time{}, false)
	bestBid, err := unsafe.GetBestBid()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if bestBid != 2 {
		t.Fatal("unexpected value")
	}
}

func TestGetBestAsk(t *testing.T) {
	t.Parallel()
	d := NewDepth(unsafeID)
	unsafe := d.GetUnsafe()
	if _, err := unsafe.GetBestAsk(); !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	d.LoadSnapshot(nil, []Item{{Price: 2}}, 0, time.Time{}, false)
	bestAsk, err := unsafe.GetBestAsk()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if bestAsk != 2 {
		t.Fatal("unexpected value")
	}
}

func TestGetMidPrice(t *testing.T) {
	t.Parallel()
	d := NewDepth(unsafeID)
	unsafe := d.GetUnsafe()
	if _, err := unsafe.GetMidPrice(); !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	d.LoadSnapshot([]Item{{Price: 1}}, []Item{{Price: 2}}, 0, time.Time{}, false)
	mid, err := unsafe.GetMidPrice()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mid != 1.5 {
		t.Fatal("unexpected value")
	}
}

func TestGetSpread(t *testing.T) {
	t.Parallel()
	d := NewDepth(unsafeID)
	unsafe := d.GetUnsafe()
	if _, err := unsafe.GetSpread(); !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	d.LoadSnapshot([]Item{{Price: 1}}, []Item{{Price: 2}}, 0, time.Time{}, false)
	spread, err := unsafe.GetSpread()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if spread != 1 {
		t.Fatal("unexpected value")
	}
}

func TestGetImbalance(t *testing.T) {
	t.Parallel()
	d := NewDepth(unsafeID)
	unsafe := d.GetUnsafe()
	_, err := unsafe.GetImbalance()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	// unlikely event zero amounts
	d.LoadSnapshot([]Item{{Price: 1, Amount: 0}}, []Item{{Price: 2, Amount: 0}}, 0, time.Time{}, false)
	_, err = unsafe.GetImbalance()
	if !errors.Is(err, errNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoLiquidity)
	}

	// balance skewed to asks
	d.LoadSnapshot([]Item{{Price: 1, Amount: 1}}, []Item{{Price: 2, Amount: 1000}}, 0, time.Time{}, false)
	imbalance, err := unsafe.GetImbalance()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if imbalance != -0.998001998001998 {
		t.Fatal("unexpected value")
	}

	// balance skewed to bids
	d.LoadSnapshot([]Item{{Price: 1, Amount: 1000}}, []Item{{Price: 2, Amount: 1}}, 0, time.Time{}, false)
	imbalance, err = unsafe.GetImbalance()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if imbalance != 0.998001998001998 {
		t.Fatal("unexpected value")
	}

	// in balance
	d.LoadSnapshot([]Item{{Price: 1, Amount: 1}}, []Item{{Price: 2, Amount: 1}}, 0, time.Time{}, false)
	imbalance, err = unsafe.GetImbalance()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if imbalance != 0 {
		t.Fatal("unexpected value")
	}
}

func TestIsStreaming(t *testing.T) {
	d := NewDepth(unsafeID)
	unsafe := d.GetUnsafe()
	if !unsafe.IsStreaming() {
		t.Fatalf("received: '%v' but expected: '%v'", unsafe.IsStreaming(), true)
	}

	d.LoadSnapshot([]Item{{Price: 1, Amount: 1}}, []Item{{Price: 2, Amount: 1}}, 0, time.Time{}, true)
	if unsafe.IsStreaming() {
		t.Fatalf("received: '%v' but expected: '%v'", unsafe.IsStreaming(), false)
	}

	d.LoadSnapshot([]Item{{Price: 1, Amount: 1}}, []Item{{Price: 2, Amount: 1}}, 0, time.Time{}, false)
	if !unsafe.IsStreaming() {
		t.Fatalf("received: '%v' but expected: '%v'", unsafe.IsStreaming(), true)
	}
}
