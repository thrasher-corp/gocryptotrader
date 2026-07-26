package orderbook

import (
	"testing"
	"time"
)

// Reproduce with identical copies of this file in separate worktrees at baseline
// 301d9636271013e697dba8ad0f9c4627ef7e6286 and candidate
// baa730ca9848beaf51541b52530c745f8d2c0cc6 (Go 1.26.1, linux/amd64 WSL2,
// Intel i7-6700K, GOMAXPROCS=1, warm Go build cache). Build each harness with:
// GOMAXPROCS=1 go test -c ./exchanges/orderbook -o orderbook.test
// Alternate 20 baseline/candidate single-sample executions of each command, appending
// their outputs to baseline.txt and candidate.txt:
// GOMAXPROCS=1 ./orderbook.test -test.run '^$' -test.bench '^BenchmarkInsertUpdates/<case>$' -test.benchmem -test.benchtime=20000x -test.count=1
// Run <case> separately for MiddleFullCapacity, MiddleSpareCapacity, TailSpareCapacity,
// and Collision.
// GOMAXPROCS=1 ./orderbook.test -test.run '^$' -test.bench '^BenchmarkProcessUpdateInsertDelete$' -test.benchmem -test.benchtime=100000x -test.count=1
// Compare the raw outputs with: benchstat baseline.txt candidate.txt
//
// Benchstat medians, n=20 per commit:
// MiddleFullCapacity: 12.075 us/op, 32640 B/op, 2 allocs/op -> 8.334 us/op, 21760 B/op, 1 alloc/op (-30.98%, p=0.000)
// MiddleSpareCapacity: 5942.5 ns/op, 10880 B/op, 1 alloc/op -> 476.1 ns/op, 0 B/op, 0 allocs/op (-91.99%, p=0.000)
// TailSpareCapacity: 621.0 ns/op -> 533.2 ns/op (-14.14%, p=0.001); both 0 B/op, 0 allocs/op
// Collision: 632.7 ns/op -> 618.9 ns/op (p=0.341, not significant); both 104 B/op, 3 allocs/op
// ProcessUpdateInsertDelete: 6271.0 ns/op, 10880 B/op, 1 alloc/op -> 752.8 ns/op, 0 B/op, 0 allocs/op (-88.00%, p=0.000)
func BenchmarkInsertUpdates(b *testing.B) {
	const (
		levelCount = 256
		middle     = levelCount / 2
	)

	snapshot := make(Levels, levelCount)
	for i := range snapshot {
		snapshot[i] = Level{Price: float64(i * 2), Amount: 1, ID: int64(i + 1)}
	}
	middleUpdate := Levels{{Price: float64(middle*2 - 1), Amount: 2, ID: levelCount + 1}}

	b.Run("MiddleFullCapacity", func(b *testing.B) {
		for b.Loop() {
			levels := append(Levels(nil), snapshot...)
			if err := levels.insertUpdates(middleUpdate, askCompare); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("MiddleSpareCapacity", func(b *testing.B) {
		levels := make(Levels, len(snapshot), len(snapshot)+1)
		copy(levels, snapshot)
		for b.Loop() {
			if err := levels.insertUpdates(middleUpdate, askCompare); err != nil {
				b.Fatal(err)
			}
			copy(levels[middle:], levels[middle+1:])
			levels = levels[:len(snapshot)]
		}
	})

	b.Run("TailSpareCapacity", func(b *testing.B) {
		levels := make(Levels, len(snapshot), len(snapshot)+1)
		copy(levels, snapshot)
		update := Levels{{Price: float64(levelCount * 2), Amount: 2, ID: levelCount + 1}}
		for b.Loop() {
			if err := levels.insertUpdates(update, askCompare); err != nil {
				b.Fatal(err)
			}
			levels = levels[:len(snapshot)]
		}
	})

	b.Run("Collision", func(b *testing.B) {
		levels := make(Levels, len(snapshot))
		copy(levels, snapshot)
		update := Levels{{Price: snapshot[middle].Price, Amount: 2, ID: levelCount + 1}}
		for b.Loop() {
			if err := levels.insertUpdates(update, askCompare); err == nil {
				b.Fatal("insertUpdates should return a collision error")
			}
		}
	})
}

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
