package orderbook

import (
	"testing"

	"github.com/gofrs/uuid"
)

var unsafeID, _ = uuid.NewV4()

func TestUnsafe(t *testing.T) {
	d := newDepth(unsafeID)
	ob := d.GetUnsafe()
	if ob.AskHead == nil || ob.BidHead == nil || ob.M == nil {
		t.Fatal("these items should not be nil")
	}
}
