package orderbookmanager

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// BenchmarkLoadSnapshotExistingDepth guards the shared-lock fast path against accidental serialisation.
func BenchmarkLoadSnapshotExistingDepth(b *testing.B) {
	relay := stream.NewRelay(1)
	pair := currency.NewBTCUSD()
	book := &orderbook.Book{
		Exchange:    "BenchmarkLoadSnapshotExistingDepth",
		Pair:        pair,
		Asset:       asset.Spot,
		LastUpdated: time.Unix(1, 0),
	}
	ob := &Orderbook{
		exchangeName: book.Exchange,
		ob:           make(map[key.PairAsset]*orderbook.Depth),
		dataHandler:  relay,
	}
	if err := ob.LoadSnapshot(b.Context(), book); err != nil {
		b.Fatal(err)
	}
	if !relayed(relay, 1) {
		b.Fatal("LoadSnapshot must relay the book")
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := ob.LoadSnapshot(b.Context(), book); err != nil {
			b.Fatal(err)
		}
		if !relayed(relay, 1) {
			b.Fatal("LoadSnapshot must relay the book")
		}
	}
}

// BenchmarkApplyPendingUpdates times LoadSnapshot, applyPendingUpdates and the relay drain together.
func BenchmarkApplyPendingUpdates(b *testing.B) {
	for _, updateCount := range []uint{1, 2, 8, 64, 256} {
		benchmarkName := strconv.FormatUint(uint64(updateCount), 10)
		b.Run(benchmarkName, func(b *testing.B) {
			// Reserve one payload for the snapshot and one for each update.
			relay := stream.NewRelay(updateCount + 1)
			exchangeName := "Benchmark-" + benchmarkName

			ob := &Orderbook{
				exchangeName: exchangeName,
				ob:           make(map[key.PairAsset]*orderbook.Depth),
				dataHandler:  relay,
			}
			manager := NewUpdateManager(&UpdateManagerParams{
				FetchDeadline: time.Second,
				FetchOrderbook: func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
					return nil, orderbook.ErrDepthNotFound
				},
				CheckPendingUpdate: func(_, _ int64, _ *orderbook.Update) (bool, error) {
					return false, nil
				},
				Orderbook: ob,
			})
			pair := currency.NewBTCUSD()
			now := time.Unix(1, 0)
			book := &orderbook.Book{
				Exchange:     exchangeName,
				Pair:         pair,
				Asset:        asset.Spot,
				LastUpdated:  now,
				LastUpdateID: 0,
			}
			updates := make([]pendingUpdate, updateCount)
			for i := range updates {
				updateID := int64(i + 1)
				updates[i] = pendingUpdate{
					firstUpdateID: updateID,
					update: &orderbook.Update{
						Pair:       pair,
						Asset:      asset.Spot,
						UpdateID:   updateID,
						UpdateTime: now,
						AllowEmpty: true,
					},
				}
			}
			cache := updateCache{updates: updates}

			if err := ob.LoadSnapshot(b.Context(), book); err != nil {
				b.Fatal(err)
			}
			cache.state = cacheStateQueuing
			if err := manager.applyPendingUpdates(&cache); err != nil {
				b.Fatal(err)
			}
			if cache.state != cacheStateSynced {
				b.Fatalf("applyPendingUpdates cache state = %d, want %d", cache.state, cacheStateSynced)
			}
			lastUpdateID, err := ob.LastUpdateID(pair, asset.Spot)
			if err != nil {
				b.Fatal(err)
			}
			expectedLastUpdateID := updates[len(updates)-1].update.UpdateID
			if lastUpdateID != expectedLastUpdateID {
				b.Fatalf("LastUpdateID = %d, want %d", lastUpdateID, expectedLastUpdateID)
			}
			if !relayed(relay, updateCount+1) {
				b.Fatal("every snapshot and update must be relayed")
			}

			b.ReportAllocs()
			for b.Loop() {
				if err := ob.LoadSnapshot(b.Context(), book); err != nil {
					b.Fatal(err)
				}
				cache.state = cacheStateQueuing
				if err := manager.applyPendingUpdates(&cache); err != nil {
					b.Fatal(err)
				}
				if !relayed(relay, updateCount+1) {
					b.Fatal("every snapshot and update must be relayed")
				}
			}
		})
	}
}

// relayed receives n payloads from relay, reporting false if any is missing, as a closed and drained relay
// has none. Relay.Send queues its payload before returning or else fails, so a payload missing after the
// calls succeeded was never sent, and failing then keeps a regression from hanging the benchmark, since go
// test's -timeout does not cover benchmarks.
func relayed(relay *stream.Relay, n uint) bool {
	for range n {
		select {
		case _, ok := <-relay.C:
			if !ok {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func TestRelayed(t *testing.T) {
	t.Parallel()
	relay := stream.NewRelay(2)
	for range 2 {
		require.NoError(t, relay.Send(t.Context(), struct{}{}), "Send must not error")
	}
	assert.True(t, relayed(relay, 2), "relayed should receive every queued payload")
	assert.Empty(t, relay.C, "relayed should consume the payloads it receives")
	assert.False(t, relayed(relay, 1), "relayed should report a payload that was never sent")

	require.NoError(t, relay.Send(t.Context(), struct{}{}), "Send must not error")
	assert.False(t, relayed(relay, 2), "relayed should report a payload missing after one that was sent")

	relay.Close()
	assert.False(t, relayed(relay, 1), "relayed should not count a closed relay as a payload")
}
