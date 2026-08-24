package orderbook

import (
	"math"
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

// BenchmarkSortAsksDescending measures the full-sort path for asks.
func BenchmarkSortAsksDescending(b *testing.B) {
	lvls := levelsFixture()
	lvls.Reverse()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortAsks()
	}
}

// BenchmarkSortBidsAscending measures the full-sort path for bids.
func BenchmarkSortBidsAscending(b *testing.B) {
	lvls := levelsFixture()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortBids()
	}
}

// BenchmarkSortAsksLateInversion measures the fallback after a full ordered-prefix scan.
func BenchmarkSortAsksLateInversion(b *testing.B) {
	lvls := levelsFixture()
	lvls[len(lvls)-1].Price = 0.5
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortAsks()
	}
}

// BenchmarkSortBidsLateInversion measures the fallback after a full ordered-prefix scan.
func BenchmarkSortBidsLateInversion(b *testing.B) {
	lvls := levelsFixture()
	lvls.Reverse()
	lvls[len(lvls)-1].Price = float64(len(lvls)) + 0.5
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortBids()
	}
}

// BenchmarkSortAsksNaN measures each legacy NaN fallback entry point for asks.
func BenchmarkSortAsksNaN(b *testing.B) {
	early := levelsFixture()
	early[0].Price = math.NaN()
	late := levelsFixture()
	late[len(late)-1].Price = math.NaN()
	afterInversion := levelsFixture()
	afterInversion[0], afterInversion[1] = afterInversion[1], afterInversion[0]
	afterInversion[len(afterInversion)-1].Price = math.NaN()

	for _, tc := range []struct {
		name   string
		levels Levels
	}{
		{name: "early", levels: early},
		{name: "late", levels: late},
		{name: "after-inversion", levels: afterInversion},
	} {
		b.Run(tc.name, func(b *testing.B) {
			bucket := make(Levels, len(tc.levels))
			for b.Loop() {
				copy(bucket, tc.levels)
				bucket.SortAsks()
			}
		})
	}
}

// BenchmarkSortBidsNaN measures each legacy NaN fallback entry point for bids.
func BenchmarkSortBidsNaN(b *testing.B) {
	early := levelsFixture()
	early.Reverse()
	early[0].Price = math.NaN()
	late := levelsFixture()
	late.Reverse()
	late[len(late)-1].Price = math.NaN()
	afterInversion := levelsFixture()
	afterInversion.Reverse()
	afterInversion[0], afterInversion[1] = afterInversion[1], afterInversion[0]
	afterInversion[len(afterInversion)-1].Price = math.NaN()

	for _, tc := range []struct {
		name   string
		levels Levels
	}{
		{name: "early", levels: early},
		{name: "late", levels: late},
		{name: "after-inversion", levels: afterInversion},
	} {
		b.Run(tc.name, func(b *testing.B) {
			bucket := make(Levels, len(tc.levels))
			for b.Loop() {
				copy(bucket, tc.levels)
				bucket.SortBids()
			}
		})
	}
}

func BenchmarkSortAsksStandard(b *testing.B) {
	lvls := levelsFixtureRandom()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortAsks()
	}
}

func BenchmarkSortBidsStandard(b *testing.B) {
	lvls := levelsFixtureRandom()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortBids()
	}
}

func BenchmarkSortAsksAscending(b *testing.B) {
	lvls := levelsFixture()
	bucket := make(Levels, len(lvls))
	for b.Loop() {
		copy(bucket, lvls)
		bucket.SortAsks()
	}
}

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
