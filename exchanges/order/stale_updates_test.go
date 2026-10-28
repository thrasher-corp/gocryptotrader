package order

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestUpdateOrderFromDetailDerivedExecution(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                               string
		amount, left, amended, amendedLeft float64
		modified, keepsStored              bool
	}{
		{"amended down", 10, 9.7, 5, 4.7, false, false},
		{"amended up", 232.31, 204.41, 7981.87, 7953.97, false, true},
		{"amended well down", 4.2, 4.1, 0.9, 0.8, false, false},
		{"modified well down", 16.1, 15.2, 1, 0.1, true, false},
		{"modified up", 0.2, 0.1, 0.3, 0.2, true, true},
		{"amended down, earlier execution rounds higher", 12.21, 8.87, 9.79, 6.45, false, false},
		{"amended down a thousandfold", 1000, 999.9, 1, 0.9, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stored := Detail{Amount: tc.amount, ExecutedAmount: tc.amount - tc.left, RemainingAmount: tc.left, Status: PartiallyFilled}
			if tc.modified {
				stored.UpdateOrderFromModifyResponse(&ModifyResponse{Amount: tc.amended})
			}
			for _, status := range []Status{PartiallyFilled, Cancelled} {
				update := Detail{Amount: tc.amended, ExecutedAmount: tc.amended - tc.amendedLeft, RemainingAmount: tc.amendedLeft, Status: status}
				original := update.Copy()
				for range 2 {
					require.NoError(t, stored.UpdateOrderFromDetail(&update), "amended order update must merge on both ingestion passes")
				}
				assert.Equal(t, original, update, "rounding adjustment should not mutate the incoming order")
			}
			stale := Detail{Amount: tc.amount, ExecutedAmount: tc.amount - tc.left, RemainingAmount: tc.left, Status: PartiallyFilled}
			require.NoError(t, stored.UpdateOrderFromDetail(&stale), "snapshot from before the amendment must not error")
			assert.Equal(t, Cancelled, stored.Status, "cancellation should finish the amended order")
			assert.Equal(t, tc.amended, stored.Amount, "amendment should replace the amount")
			assert.Equal(t, tc.amendedLeft, stored.RemainingAmount, "amendment should replace the remaining quantity")
			executed := tc.amended - tc.amendedLeft
			if tc.keepsStored {
				executed = tc.amount - tc.left
			}
			assert.Equal(t, executed, stored.ExecutedAmount, "execution should follow updates derived from smaller quantities")
		})
	}
}

func TestUpdateOrderFromModifyResponseReplacement(t *testing.T) {
	t.Parallel()
	filled := func(orderID string) Detail {
		return Detail{OrderID: orderID, Amount: 10, ExecutedAmount: 6, RemainingAmount: 4, ExecutedQuoteAmount: 600, AverageExecutedPrice: 100, Fee: 0.6, FeeAsset: currency.AUD, Trades: []TradeHistory{{TID: "t-1", Price: 100, Amount: 6}}, Status: PartiallyFilled}
	}
	execution := func(d *Detail) []any {
		return []any{d.ExecutedAmount, d.ExecutedQuoteAmount, d.AverageExecutedPrice, d.Fee, d.FeeAsset, d.Trades}
	}
	for _, storedID := range []string{"1", ""} {
		d := filled(storedID)
		d.UpdateOrderFromModifyResponse(&ModifyResponse{OrderID: "1", Amount: 12})
		want := filled(storedID)
		assert.Equalf(t, execution(&want), execution(&d), "an amendment of order %q in place should keep its execution", storedID)
	}
	d := filled("1")
	d.UpdateOrderFromModifyResponse(&ModifyResponse{OrderID: "2", Amount: 10, RemainingAmount: 9})
	assert.Equal(t, []any{0.0, 0.0, 0.0, 0.0, currency.EMPTYCODE, []TradeHistory(nil)}, execution(&d), "a replacement order should not inherit the replaced order's execution")
	require.NoError(t, d.UpdateOrderFromDetail(&Detail{OrderID: "2", Amount: 10, ExecutedAmount: 1, RemainingAmount: 9, Status: Cancelled}), "replacement order update must merge")
	assert.Equal(t, Cancelled, d.Status, "the replacement order's cancellation should finish it")
	assert.Equal(t, 1.0, d.ExecutedAmount, "the replacement order's execution should be its own")
}

func TestUpdateOrderFromDetailAmendmentSequences(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(2375, 5665619267)) //nolint:gosec // Fixed seeds reproduce test quantities; this is not security-sensitive randomness.
	for sequence := range 2000 {
		filled := rng.Int64N(9999) + 1
		total := filled + rng.Int64N(100000) + 1
		amount, left := float64(total)/100, float64(total-filled)/100
		stored := Detail{Amount: amount, RemainingAmount: left, ExecutedAmount: amount - left, Status: PartiallyFilled}
		for step := range 4 {
			total = filled + rng.Int64N(100000) + 1
			amount, left = float64(total)/100, float64(total-filled)/100
			if step == 1 {
				stored.UpdateOrderFromModifyResponse(&ModifyResponse{Amount: amount})
			}
			status := PartiallyFilled
			if step == 3 {
				status = Cancelled
			}
			incoming := Detail{Amount: amount, RemainingAmount: left, ExecutedAmount: amount - left, Status: status}
			for range 2 {
				require.NoError(t, stored.UpdateOrderFromDetail(&incoming), "amendment sequence must merge through both ingestion passes")
			}
		}
		assert.Equalf(t, Cancelled, stored.Status, "sequence %d should finish after quantity amendments without new fills", sequence)
		assert.Equalf(t, amount, stored.Amount, "sequence %d should retain the last amended quantity", sequence)
		assert.Equalf(t, left, stored.RemainingAmount, "sequence %d should retain the last unexecuted quantity", sequence)
	}
}

func TestUpdateOrderFromDetailMixedClocks(t *testing.T) {
	t.Parallel()
	now := time.Unix(1735720640, 800000000)
	for _, offset := range []time.Duration{-800 * time.Millisecond, -500 * time.Millisecond, -30 * time.Millisecond, 0, time.Second} {
		t.Run(offset.String(), func(t *testing.T) {
			t.Parallel()
			stored := Detail{Amount: 3, ExecutedAmount: 2, Status: PartiallyFilled, Fee: 0.6, LastUpdated: now}
			incoming := Detail{Amount: 3, ExecutedAmount: 2, Status: Cancelled, Fee: 1.2, CloseTime: now.Add(offset), LastUpdated: now.Add(offset)}
			require.NoError(t, stored.UpdateOrderFromDetail(&incoming), "venue cancellation must merge despite mixed timestamp origins")
			assert.Equal(t, Cancelled, stored.Status, "venue cancellation should finish the order")
			assert.Equal(t, 1.2, stored.Fee, "venue cancellation should preserve the full cumulative fee")
			assert.Equal(t, incoming.CloseTime, stored.CloseTime, "venue cancellation should preserve its close time")
			assert.False(t, stored.LastUpdated.Before(now), "display update time should not move backwards")
		})
	}
	for _, status := range []Status{Open, PartiallyFilled} {
		stored := Detail{Amount: 3, ExecutedAmount: 2, Status: status, LastUpdated: now}
		require.NoError(t, stored.UpdateOrderFromDetail(&Detail{Status: Cancelled, LastUpdated: now.Truncate(time.Second)}), "status-only cancellation must merge")
		assert.Equal(t, Cancelled, stored.Status, "status-only cancellation should finish an open or partially filled order")
		assert.Equal(t, 2.0, stored.ExecutedAmount, "status-only cancellation should retain execution")
	}
	stored := Detail{Amount: 0.2, ExecutedAmount: 0.2 - 0.1, Status: PartiallyFilled, LastUpdated: now}
	amount, remaining := 0.3, 0.2
	require.NoError(t, stored.UpdateOrderFromDetail(&Detail{Amount: amount, ExecutedAmount: amount - remaining, RemainingAmount: remaining, Status: PartiallyFilled, LastUpdated: now.Truncate(time.Second)}), "rounded equal-fill amendment must merge")
	assert.Equal(t, 0.3, stored.Amount, "rounding should not suppress an amended amount")
	assert.Equal(t, 0.2, stored.RemainingAmount, "rounding should not suppress amended remaining quantity")
	assert.Equal(t, 0.1, stored.ExecutedAmount, "rounding should not rewind cumulative execution")
	for _, filled := range []float64{1e-12, 0.1, 1e12} {
		progress := Detail{ExecutedAmount: filled, Amount: filled * 2, Status: Cancelled}
		next := math.Nextafter(filled, math.Inf(1))
		require.NoError(t, progress.UpdateOrderFromDetail(&Detail{ExecutedAmount: next, Status: PartiallyFilled}), "one-step execution increase after cancellation must not error")
		assert.Equal(t, filled, progress.ExecutedAmount, "one representable step after cancellation should count as float rounding")
		for range 128 {
			next = math.Nextafter(next, math.Inf(1))
		}
		require.NoError(t, progress.UpdateOrderFromDetail(&Detail{ExecutedAmount: next, Status: PartiallyFilled}), "small positive progress must merge")
		assert.Equal(t, next, progress.ExecutedAmount, "positive execution beyond float rounding should be retained")
		assert.Equal(t, Cancelled, progress.Status, "small positive progress should not reopen the order")
		stored := Detail{ExecutedAmount: filled, Amount: filled * 2, Status: PartiallyFilled}
		before := stored.Copy()
		lower := filled
		for range 128 {
			lower = math.Nextafter(lower, 0)
		}
		require.NoError(t, stored.UpdateOrderFromDetail(&Detail{ExecutedAmount: lower, Amount: filled * 3, Status: Open}), "decreased fill snapshot must not error")
		assert.Equal(t, before, stored, "a decrease beyond rounding should remain stale at every quantity scale")
	}
	for _, status := range []Status{Open, PartiallyFilled, PendingCancel, Cancelling} {
		for _, filled := range []float64{2, 3} {
			stored := Detail{Amount: 3, ExecutedAmount: 2, Status: Cancelled}
			require.NoError(t, stored.UpdateOrderFromDetail(&Detail{ExecutedAmount: filled, Status: status}), "post-cancellation update must merge safely")
			assert.Equal(t, Cancelled, stored.Status, "fill progress should never reopen a terminal order")
			assert.Equal(t, filled, stored.ExecutedAmount, "fill progress should still be retained after cancellation")
		}
	}
}

func TestUpdateOrderFromDetailDelayedSnapshots(t *testing.T) {
	t.Parallel()
	now := time.Unix(1735720640, 0)
	for _, timestamp := range []time.Time{now.Add(-time.Second), now, now.Add(time.Second), {}} {
		t.Run(timestamp.String(), func(t *testing.T) {
			t.Parallel()
			stored := Detail{ExecutedAmount: 2, ExecutedQuoteAmount: 120, AverageExecutedPrice: 60, Fee: 0.2, RemainingAmount: 1, Amount: 3, Status: PartiallyFilled, AssetType: asset.Spot, LastUpdated: now, Trades: []TradeHistory{{TID: "known", Price: 60}}}
			before := stored.Copy()
			incoming := Detail{ExecutedAmount: 1, ExecutedQuoteAmount: 50, AverageExecutedPrice: 50, Fee: 0.1, RemainingAmount: 2, Amount: 3, Status: Open, AssetType: asset.Spot, LastUpdated: timestamp, Trades: []TradeHistory{{TID: "known", Price: 50}, {TID: "unseen", Price: 55}}}
			original := incoming.Copy()
			require.NoError(t, stored.UpdateOrderFromDetail(&incoming), "delayed snapshot must not error")
			before.Trades = append(before.Trades, TradeHistory{TID: "unseen", Price: 55})
			assert.Equal(t, before, stored, "delayed snapshot should retain newer state and only add unseen fills")
			assert.Equal(t, original, incoming, "merge should not mutate the incoming snapshot")
			require.NoError(t, stored.UpdateOrderFromDetail(&incoming), "replayed stale snapshot must not error")
			assert.Equal(t, before, stored, "replayed stale snapshot should be idempotent")
		})
	}
	stored := Detail{ExecutedAmount: 2, AverageExecutedPrice: 60, Status: Filled, LastUpdated: now}
	require.NoError(t, stored.UpdateOrderFromDetail(&Detail{ExecutedAmount: 2, AverageExecutedPrice: 50, Status: Open, LastUpdated: now.Add(-time.Second)}), "older same-fill snapshot must not error")
	assert.Equal(t, 60.0, stored.AverageExecutedPrice, "same-fill older snapshot should not replace the average")
	assert.Equal(t, Filled, stored.Status, "older snapshot should not reopen a finished order")
	require.NoError(t, stored.UpdateOrderFromDetail(&Detail{Status: Open, LastUpdated: now.Add(-time.Second)}), "older status-only snapshot must not error")
	assert.Equal(t, Filled, stored.Status, "older status-only snapshot should not reopen the order")
	require.NoError(t, stored.UpdateOrderFromDetail(&Detail{ExecutedAmount: 3, AverageExecutedPrice: 61, LastUpdated: now.Add(-time.Second)}), "higher cumulative execution must not be lost to timestamp skew")
	assert.Equal(t, 3.0, stored.ExecutedAmount, "higher execution should advance despite the earlier timestamp")
	assert.Equal(t, 61.0, stored.AverageExecutedPrice, "higher execution should retain its associated average")
	assert.Equal(t, now, stored.LastUpdated, "progress with an earlier timestamp should not rewind LastUpdated")
	require.NoError(t, stored.UpdateOrderFromDetail(&Detail{ExecutedAmount: 4, AverageExecutedPrice: 62, LastUpdated: now.Add(time.Second)}), "newer cumulative progress must merge")
	assert.Equal(t, 4.0, stored.ExecutedAmount, "newer cumulative execution should advance")
	assert.Equal(t, now.Add(time.Second), stored.LastUpdated, "newer progress should advance LastUpdated")
	require.NoError(t, stored.UpdateOrderFromDetail(&Detail{Fee: 0.3, LastUpdated: now.Add(time.Second)}), "same-timestamp enrichment must merge")
	assert.Equal(t, 0.3, stored.Fee, "same-timestamp enrichment should retain reported metadata")
	assert.Equal(t, now.Add(time.Second), stored.LastUpdated, "same-timestamp enrichment should not replace exchange time with the local clock")
}
