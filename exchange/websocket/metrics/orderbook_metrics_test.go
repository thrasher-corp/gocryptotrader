package metrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestRecordOrderbookDesync(t *testing.T) {
	t.Parallel()

	event := OrderbookSyncEvent{
		Exchange:      "MetricDesyncExchange",
		Pair:          currency.NewBTCUSDT(),
		Asset:         asset.Spot,
		Channel:       "books",
		Reason:        "sequence_gap",
		LastUpdateID:  10,
		FirstUpdateID: 14,
		UpdateID:      15,
	}
	key := formatOrderbookSyncMetricKey(&event)

	RecordOrderbookDesync(&event)

	assert.Contains(t, orderbookDesyncMetric.String(), `"`+key+`": 1`, "desync metric should increment for the event key")
	assert.Contains(t, orderbookDesyncGapMetric.String(), `"`+key+`": 3`, "desync gap metric should record skipped sequence count")
}

func TestRecordOrderbookDesyncNil(t *testing.T) {
	t.Parallel()

	RecordOrderbookDesync(nil)
}

func TestRecordOrderbookResync(t *testing.T) {
	t.Parallel()

	event := OrderbookSyncEvent{
		Exchange: "MetricResyncExchange",
		Pair:     currency.NewPair(currency.ETH, currency.USDT),
		Asset:    asset.PerpetualSwap,
		Channel:  "books",
		Reason:   "checksum",
		Result:   "requested",
	}
	key := formatOrderbookSyncMetricKey(&event)

	RecordOrderbookResync(&event)

	assert.Contains(t, orderbookResyncMetric.String(), `"`+key+`": 1`, "resync metric should increment for the event key")
}

func TestRecordOrderbookResyncNil(t *testing.T) {
	t.Parallel()

	RecordOrderbookResync(nil)
}

func TestSnapshotOrderbookSyncSummary(t *testing.T) {
	pair := currency.NewPair(currency.SOL, currency.USDT)
	event := &OrderbookSyncEvent{
		Exchange:      "MetricSnapshotExchange",
		Pair:          pair,
		Asset:         asset.Spot,
		Channel:       "books",
		Reason:        "sequence_gap",
		LastUpdateID:  20,
		FirstUpdateID: 23,
	}
	RecordOrderbookDesync(event)
	RecordOrderbookResync(&OrderbookSyncEvent{
		Exchange: "MetricSnapshotExchange",
		Pair:     pair,
		Asset:    asset.Spot,
		Channel:  "books",
		Result:   "started",
	})

	summary := SnapshotOrderbookSyncSummary()

	var found bool
	for _, item := range summary.Items {
		if item.Exchange != "MetricSnapshotExchange" || item.Pair != "SOLUSDT" {
			continue
		}
		found = true
		assert.Equal(t, int64(1), item.Desyncs, "desync count should match")
		assert.Equal(t, int64(2), item.SequenceGapTotal, "sequence gap total should match")
		assert.Equal(t, int64(1), item.DroppedOrderbooks, "dropped orderbook count should match")
		assert.Equal(t, int64(1), item.ResyncStarted, "resync started count should match")
		assert.Equal(t, int64(1), item.Reasons["sequence_gap"], "reason count should match")
	}
	assert.True(t, found, "summary should include recorded item")
}

func TestOrderbookSyncSummaryLines(t *testing.T) {
	pair := currency.NewPair(currency.ADA, currency.USDT)
	for range 20 {
		RecordOrderbookDesync(&OrderbookSyncEvent{
			Exchange:      "MetricSummaryExchange",
			Pair:          pair,
			Asset:         asset.Margin,
			Channel:       "books",
			Reason:        "checksum",
			LastUpdateID:  30,
			FirstUpdateID: 32,
		})
		RecordOrderbookResync(&OrderbookSyncEvent{
			Exchange: "MetricSummaryExchange",
			Pair:     pair,
			Asset:    asset.Margin,
			Channel:  "books",
			Reason:   "checksum",
			Result:   "requested",
		})
	}

	lines := OrderbookSyncSummaryLines()

	require.NotEmpty(t, lines, "summary lines must be produced after metrics are recorded")
	assert.Contains(t, lines[0], "Websocket orderbook sync summary:", "first line should contain summary heading")
	assert.Contains(t, strings.Join(lines, "\n"), "Worst dropped orderbook offender: exchange=MetricSummaryExchange pair=ADAUSDT asset=margin channel=books", "summary should name the worst offender")
	assert.Contains(t, strings.Join(lines, "\n"), "reasons=checksum:20", "summary should include readable reason counts")
}

func TestFormatOrderbookSyncMetricKey(t *testing.T) {
	t.Parallel()

	event := OrderbookSyncEvent{
		Exchange:      "MetricKeyExchange",
		Pair:          currency.NewPair(currency.LTC, currency.USDT),
		Asset:         asset.Margin,
		Channel:       "books",
		Reason:        "snapshot_outdated",
		Result:        "started",
		LastUpdateID:  10,
		FirstUpdateID: 20,
		UpdateID:      21,
	}

	assert.Equal(t,
		"exchange=MetricKeyExchange,asset=margin,pair=LTCUSDT,channel=books,reason=snapshot_outdated,result=started",
		formatOrderbookSyncMetricKey(&event),
		"metric key should include stable dimensions only")
}

func TestRecordOrderbookSyncDesync(t *testing.T) {
	event := &OrderbookSyncEvent{
		Exchange: "MetricRecordDesyncExchange",
		Pair:     currency.NewPair(currency.DOT, currency.USDT),
		Asset:    asset.Spot,
		Channel:  "books",
		Reason:   "snapshot_outdated",
	}

	recordOrderbookSyncDesync(event, 7)

	summary := SnapshotOrderbookSyncSummary()
	var found bool
	for _, item := range summary.Items {
		if item.Exchange != "MetricRecordDesyncExchange" {
			continue
		}
		found = true
		assert.Equal(t, int64(1), item.Desyncs, "desync count should increment")
		assert.Equal(t, int64(7), item.SequenceGapTotal, "gap count should increment")
		assert.Equal(t, int64(1), item.Reasons["snapshot_outdated"], "reason count should increment")
	}
	assert.True(t, found, "summary should include direct desync record")
}

func TestRecordOrderbookSyncResync(t *testing.T) {
	testCases := []struct {
		name   string
		result string
		assert func(*testing.T, OrderbookSyncSummaryItem)
	}{
		{
			name:   "started",
			result: "started",
			assert: func(t *testing.T, item OrderbookSyncSummaryItem) {
				t.Helper()
				assert.Equal(t, int64(1), item.ResyncStarted, "started count should increment")
			},
		},
		{
			name:   "requested",
			result: "requested",
			assert: func(t *testing.T, item OrderbookSyncSummaryItem) {
				t.Helper()
				assert.Equal(t, int64(1), item.ResyncRequested, "requested count should increment")
			},
		},
		{
			name:   "succeeded",
			result: "succeeded",
			assert: func(t *testing.T, item OrderbookSyncSummaryItem) {
				t.Helper()
				assert.Equal(t, int64(1), item.ResyncSucceeded, "succeeded count should increment")
			},
		},
		{
			name:   "failed",
			result: "failed",
			assert: func(t *testing.T, item OrderbookSyncSummaryItem) {
				t.Helper()
				assert.Equal(t, int64(1), item.ResyncFailed, "failed count should increment")
			},
		},
		{
			name:   "request_failed",
			result: "request_failed",
			assert: func(t *testing.T, item OrderbookSyncSummaryItem) {
				t.Helper()
				assert.Equal(t, int64(1), item.ResyncRequestFailed, "request failed count should increment")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pair := currency.NewPairWithDelimiter("REC"+strings.ToUpper(tc.name), "USDT", "-")
			recordOrderbookSyncResync(&OrderbookSyncEvent{
				Exchange: "MetricRecordResyncExchange",
				Pair:     pair,
				Asset:    asset.Spot,
				Channel:  "books",
				Result:   tc.result,
			})

			summary := SnapshotOrderbookSyncSummary()
			var found bool
			for _, item := range summary.Items {
				if item.Exchange != "MetricRecordResyncExchange" || item.Pair != pair.String() {
					continue
				}
				found = true
				tc.assert(t, item)
			}
			require.True(t, found, "summary must include direct resync record")
		})
	}
}

func TestOrderbookSyncStoreLoad(t *testing.T) {
	store := &orderbookSyncStore{lookup: make(map[orderbookSyncKey]*orderbookSyncCounts)}
	event := &OrderbookSyncEvent{
		Exchange: "MetricStoreExchange",
		Pair:     currency.NewPair(currency.XLM, currency.USDT),
		Asset:    asset.Spot,
		Channel:  "books",
	}

	first := store.load(event)
	second := store.load(event)

	require.Same(t, first, second, "load must return the same counts for the same event key")
	assert.Len(t, store.lookup, 1, "store should contain one key")
}

func TestOrderbookSyncSummaryItemLabel(t *testing.T) {
	item := &OrderbookSyncSummaryItem{
		Exchange: "MetricLabelExchange",
		Pair:     "BTCUSDT",
		Asset:    "spot",
		Channel:  "books",
	}

	assert.Equal(t, "exchange=MetricLabelExchange pair=BTCUSDT asset=spot channel=books", item.label(), "label should be readable")
}

func TestOrderbookSyncSummaryItemTopReason(t *testing.T) {
	testCases := []struct {
		name   string
		item   *OrderbookSyncSummaryItem
		reason string
	}{
		{
			name:   "none",
			item:   &OrderbookSyncSummaryItem{},
			reason: "none",
		},
		{
			name:   "highest",
			item:   &OrderbookSyncSummaryItem{Reasons: map[string]int64{"sequence_gap": 2, "checksum": 3}},
			reason: "checksum:3",
		},
		{
			name:   "tie_breaker",
			item:   &OrderbookSyncSummaryItem{Reasons: map[string]int64{"sequence_gap": 2, "checksum": 2}},
			reason: "checksum:2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.reason, tc.item.topReason(), "top reason should match")
		})
	}
}

func TestOrderbookSyncSummaryItemReasonSummary(t *testing.T) {
	testCases := []struct {
		name    string
		item    *OrderbookSyncSummaryItem
		summary string
	}{
		{
			name:    "none",
			item:    &OrderbookSyncSummaryItem{},
			summary: "none",
		},
		{
			name:    "sorted",
			item:    &OrderbookSyncSummaryItem{Reasons: map[string]int64{"sequence_gap": 2, "checksum": 3}},
			summary: "checksum:3,sequence_gap:2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.summary, tc.item.reasonSummary(), "reason summary should match")
		})
	}
}
