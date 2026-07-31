package orderbook

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// 705985	      1856 ns/op	       0 B/op	       0 allocs/op
func BenchmarkReverse(b *testing.B) {
	lvls := levelsFixture()
	if len(lvls) != 1000 {
		b.Fatal("incorrect length")
	}

	for b.Loop() {
		lvls.Reverse()
	}
}

// 361266	      3556 ns/op	      24 B/op	       1 allocs/op (old)
// 385783	      3000 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortAsksDecending(b *testing.B) {
	lvls := levelsFixture()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortAsks()
	}
}

// 266998	      4292 ns/op	      40 B/op	       2 allocs/op (old)
// 372396	      3001 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortBidsAscending(b *testing.B) {
	lvls := levelsFixture()
	lvls.Reverse()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortBids()
	}
}

// 22119	     46532 ns/op	      35 B/op	       1 allocs/op (old)
// 16233	     76951 ns/op	     167 B/op	       3 allocs/op (new)
func BenchmarkSortAsksStandard(b *testing.B) {
	lvls := levelsFixtureRandom()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortAsks()
	}
}

// 19504	     62518 ns/op	      53 B/op	       2 allocs/op (old)
// 15698	     72859 ns/op	     168 B/op	       3 allocs/op (new)
func BenchmarkSortBidsStandard(b *testing.B) {
	lvls := levelsFixtureRandom()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortBids()
	}
}

// 376708	      3559 ns/op	      24 B/op 		   1 allocs/op (old)
// 377113	      3020 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortAsksAscending(b *testing.B) {
	lvls := levelsFixture()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortAsks()
	}
}

// 262874	      4364 ns/op	      40 B/op	       2 allocs/op (old)
// 401788	      3348 ns/op	     152 B/op	       3 allocs/op (new)
func BenchmarkSortBidsDescending(b *testing.B) {
	lvls := levelsFixture()
	lvls.Reverse()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortBids()
	}
}

// 5572401	       210.9 ns/op	       0 B/op	       0 allocs/op (current)
// 3748009	       312.7 ns/op	      32 B/op	       1 allocs/op (previous)
func BenchmarkProcess(b *testing.B) {
	book := &Book{
		Pair:     currency.NewBTCUSD(),
		Asks:     make(Levels, 100),
		Bids:     make(Levels, 100),
		Exchange: "BenchmarkProcessOrderbook",
		Asset:    asset.Spot,
	}

	for b.Loop() {
		if err := book.Process(); err != nil {
			b.Fatal(err)
		}
	}
}
