package orderbook

import (
	"testing"
	"time"
)

// The measurement below used the standalone benchmark harness retained in
// f128775b2d1f66650dbd46b89106564fd2d33d6d. Restore that identical harness
// in separate worktrees at baseline 301d9636271013e697dba8ad0f9c4627ef7e6286
// and candidate baa730ca9848beaf51541b52530c745f8d2c0cc6 with:
// git show f128775b2d1f66650dbd46b89106564fd2d33d6d:exchanges/orderbook/insert_updates_bench_test.go > exchanges/orderbook/insert_updates_bench_test.go
// The environment was Go 1.26.1, linux/amd64 WSL2, Intel i7-6700K,
// GOMAXPROCS=1, with a warm Go build cache. Build each harness with:
// GOMAXPROCS=1 go test -c ./exchanges/orderbook -o orderbook.test
// Alternate 20 baseline/candidate single-sample executions of this command,
// appending their outputs to baseline.txt and candidate.txt:
// GOMAXPROCS=1 ./orderbook.test -test.run '^$' -test.bench '^BenchmarkProcessUpdateInsertDelete$' -test.benchmem -test.benchtime=100000x -test.count=1
// Compare the raw outputs with: benchstat baseline.txt candidate.txt
//
// Benchstat medians, n=20 per commit:
// ProcessUpdateInsertDelete: 6271.0 ns/op, 10880 B/op, 1 alloc/op -> 752.8 ns/op, 0 B/op, 0 allocs/op (-88.00%, p=0.000)
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
