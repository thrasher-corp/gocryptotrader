package orderbook

import "testing"

// 27906781	        42.4 ns/op	       0 B/op	       0 allocs/op (old)
// 84119028	        13.87 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkLoad(b *testing.B) {
	ts := Levels{}
	for b.Loop() {
		ts.load(ask)
	}
}

// 46043871	        25.9 ns/op	       0 B/op	       0 allocs/op (old)
// 63445401	        18.51 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkUpdateByID(b *testing.B) {
	asks := Levels{}
	asksSnapshot := Levels{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for b.Loop() {
		err := asks.updateByID(asksSnapshot)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 26724331	        44.69 ns/op	       0 B/op	       0 allocs/op
func BenchmarkDeleteByID(b *testing.B) {
	asks := Levels{}
	asksSnapshot := Levels{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for b.Loop() {
		err := asks.deleteByID(asksSnapshot, false)
		if err != nil {
			b.Fatal(err)
		}
		asks.load(asksSnapshot) // reset
	}
}

// 8384302	       150.9 ns/op	     480 B/op	       1 allocs/op
func BenchmarkRetrieve(b *testing.B) {
	asks := Levels{}
	asksSnapshot := Levels{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for b.Loop() {
		_ = asks.retrieve(6)
	}
}

// 134830672	         9.83 ns/op	       0 B/op	       0 allocs/op (old)
// 206689897	         5.761 ns/op	   0 B/op	       0 allocs/op (new)
func BenchmarkUpdateInsertByPrice_Amend(b *testing.B) {
	a := askLevels{}
	a.load(ask)

	updates := Levels{
		{
			Price:  1337, // Amend
			Amount: 2,
		},
		{
			Price:  1337, // Amend
			Amount: 1,
		},
	}

	for b.Loop() {
		a.updateInsertByPrice(updates, 0)
	}
}

// 49763002	        24.9 ns/op	       0 B/op	       0 allocs/op (old)
// 25662849	        45.32 ns/op	       0 B/op	       0 allocs/op (new)
func BenchmarkUpdateInsertByPrice_Insert_Delete(b *testing.B) {
	a := askLevels{}

	a.load(ask)

	updates := Levels{
		{
			Price:  1337.5, // Insert
			Amount: 2,
		},
		{
			Price:  1337.5, // Delete
			Amount: 0,
		},
	}

	for b.Loop() {
		a.updateInsertByPrice(updates, 0)
	}
}

// 21614455	        81.74 ns/op	       0 B/op	       0 allocs/op
func BenchmarkUpdateInsertByID_asks(b *testing.B) {
	asks := Levels{}
	asksSnapshot := Levels{
		{Price: 1, Amount: 1, ID: 1},
		{Price: 3, Amount: 1, ID: 3},
		{Price: 5, Amount: 1, ID: 5},
		{Price: 7, Amount: 1, ID: 7},
		{Price: 9, Amount: 1, ID: 9},
		{Price: 11, Amount: 1, ID: 11},
	}
	asks.load(asksSnapshot)

	for b.Loop() {
		err := asks.updateInsertByID(asksSnapshot, askCompare)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 20328886	        59.94 ns/op	       0 B/op	       0 allocs/op
func BenchmarkUpdateInsertByID_bids(b *testing.B) {
	bids := Levels{}
	bidsSnapshot := Levels{
		{Price: 0.5, Amount: 2, ID: 0},
		{Price: 1, Amount: 2, ID: 1},
		{Price: 3, Amount: 2, ID: 3},
		{Price: 12, Amount: 2, ID: 5},
		{Price: 7, Amount: 2, ID: 7},
		{Price: 9, Amount: 2, ID: 9},
		{Price: 11, Amount: 2, ID: 11},
	}
	bids.load(bidsSnapshot)

	for b.Loop() {
		err := bids.updateInsertByID(bidsSnapshot, bidCompare)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// To reproduce the measurements below, copy BenchmarkInsertUpdates verbatim
// into a temporary _test.go file in package orderbook with an import for
// "testing". Use separate worktrees at baseline
// 301d9636271013e697dba8ad0f9c4627ef7e6286 and candidate
// baa730ca9848beaf51541b52530c745f8d2c0cc6.
// The environment was Go 1.26.1, linux/amd64 WSL2, Intel i7-6700K,
// GOMAXPROCS=1, with a warm Go build cache. Build each harness with:
// GOMAXPROCS=1 go test -c ./exchanges/orderbook -o orderbook.test
// Alternate 20 baseline/candidate single-sample executions of each command,
// appending their outputs to baseline.txt and candidate.txt:
// GOMAXPROCS=1 ./orderbook.test -test.run '^$' -test.bench '^BenchmarkInsertUpdates/<case>$' -test.benchmem -test.benchtime=20000x -test.count=1
// Run <case> separately for MiddleFreshCopy, MiddleSpareCapacity,
// TailSpareCapacity, and Collision. Compare the raw outputs with:
// benchstat baseline.txt candidate.txt
//
// Benchstat medians, n=20 per commit:
// MiddleFreshCopy (recorded as MiddleFullCapacity): 12.075 us/op, 32640 B/op, 2 allocs/op -> 8.334 us/op, 21760 B/op, 1 alloc/op (-30.98%, p=0.000)
// MiddleSpareCapacity: 5942.5 ns/op, 10880 B/op, 1 alloc/op -> 476.1 ns/op, 0 B/op, 0 allocs/op (-91.99%, p=0.000)
// TailSpareCapacity: 621.0 ns/op -> 533.2 ns/op (-14.14%, p=0.001); both 0 B/op, 0 allocs/op
// Collision: 632.7 ns/op -> 618.9 ns/op (p=0.341, not significant); both 104 B/op, 3 allocs/op
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

	b.Run("MiddleFreshCopy", func(b *testing.B) {
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
