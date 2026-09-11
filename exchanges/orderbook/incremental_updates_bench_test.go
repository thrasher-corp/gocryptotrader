package orderbook

import (
	"testing"
	"time"
)

// Benchstat medians for PR base to measured production head
// (20 counterbalanced fresh-process observations per revision):
// Before: 5021.0 ns/op  10880 B/op  1 allocs/op; After: 688.1 ns/op  0 B/op  0 allocs/op
func BenchmarkProcessUpdateInsertDelete(b *testing.B) {
	depth := NewDepth(id)
	if err := depth.LoadSnapshot(newSnapshot(256)); err != nil {
		b.Fatal(err)
	}

	updateTime := time.Unix(1, 0)
	insertUpdate := &Update{
		UpdateTime: updateTime,
		Asks:       Levels{{Price: 1465.5, Amount: 2, ID: 1000}},
		Action:     InsertAction,
	}
	deleteUpdate := &Update{
		UpdateTime: updateTime,
		Asks:       Levels{{ID: 1000}},
		Action:     DeleteAction,
	}
	for b.Loop() {
		if err := depth.ProcessUpdate(insertUpdate); err != nil {
			b.Fatal(err)
		}
		if err := depth.ProcessUpdate(deleteUpdate); err != nil {
			b.Fatal(err)
		}
	}
}
