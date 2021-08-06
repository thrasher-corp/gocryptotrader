package orderbook

import (
	"testing"

	"github.com/gofrs/uuid"
)

var unsafeID, _ = uuid.NewV4()

type externalBook struct{}

func (e *externalBook) Lock()   {}
func (e *externalBook) Unlock() {}

func TestUnsafe(t *testing.T) {
	d := newDepth(unsafeID)
	ob := d.GetUnsafe()
	if ob.AskHead == nil || ob.BidHead == nil || ob.m == nil {
		t.Fatal("these items should not be nil")
	}

	ob2 := &externalBook{}
	ob.Lock()
	ob.Unlock() // nolint:staticcheck // Not needed in test
	ob.LockWith(ob2)
	ob.UnlockWith(ob2)
}
