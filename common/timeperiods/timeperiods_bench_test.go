package timeperiods

import (
	"testing"
	"time"
)

func BenchmarkFindTimeRangesContainingData(b *testing.B) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	sparseComparisonTimes := make([]time.Time, 0, 24*12)
	for offset := time.Duration(0); offset < end.Sub(start); offset += 5 * time.Minute {
		sparseComparisonTimes = append(sparseComparisonTimes, start.Add(offset+30*time.Second))
	}
	denseComparisonTimes := make([]time.Time, 0, 24*60)
	for offset := time.Duration(0); offset < end.Sub(start); offset += time.Minute {
		denseComparisonTimes = append(denseComparisonTimes, start.Add(offset+30*time.Second))
	}
	outOfRangeComparisonTimes := make([]time.Time, len(sparseComparisonTimes))
	for i := range sparseComparisonTimes {
		outOfRangeComparisonTimes[i] = sparseComparisonTimes[i].Add(-24 * time.Hour)
	}
	run := func(b *testing.B, comparisonTimes []time.Time) {
		b.Helper()
		b.ReportAllocs()
		for b.Loop() {
			_, err := FindTimeRangesContainingData(start, end, time.Minute, comparisonTimes)
			if err != nil {
				b.Fatal(err)
			}
		}
	}

	b.Run("empty", func(b *testing.B) { run(b, nil) })
	b.Run("sparse", func(b *testing.B) { run(b, sparseComparisonTimes) })
	b.Run("dense", func(b *testing.B) { run(b, denseComparisonTimes) })
	b.Run("out-of-range", func(b *testing.B) { run(b, outOfRangeComparisonTimes) })
}
