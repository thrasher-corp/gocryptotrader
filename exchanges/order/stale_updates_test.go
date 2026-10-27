package order

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

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
