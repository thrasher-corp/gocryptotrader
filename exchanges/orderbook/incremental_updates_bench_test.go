package orderbook

import (
	"testing"
	"time"
)

func BenchmarkProcessUpdateUpdateOrInsertDelete(b *testing.B) {
	depth := NewDepth(id)
	if err := depth.LoadSnapshot(newSnapshot(256)); err != nil {
		b.Fatal(err)
	}

	updateTime := time.Unix(1, 0)
	updateOrInsertUpdate := &Update{
		UpdateTime: updateTime,
		Asks:       Levels{{Price: 1465.5, Amount: 2, ID: 1000}},
		Action:     UpdateOrInsertAction,
	}
	deleteUpdate := &Update{
		UpdateTime: updateTime,
		Asks:       Levels{{ID: 1000}},
		Action:     DeleteAction,
	}
	// The first insert grows the levels once, so it comes before b.Loop to keep that out of B/op
	if err := depth.ProcessUpdate(updateOrInsertUpdate); err != nil {
		b.Fatal(err)
	}
	if err := depth.ProcessUpdate(deleteUpdate); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if err := depth.ProcessUpdate(updateOrInsertUpdate); err != nil {
			b.Fatal(err)
		}
		if err := depth.ProcessUpdate(deleteUpdate); err != nil {
			b.Fatal(err)
		}
	}
}
