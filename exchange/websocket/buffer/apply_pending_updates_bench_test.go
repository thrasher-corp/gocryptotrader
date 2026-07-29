package buffer

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

func BenchmarkApplyPendingUpdates(b *testing.B) {
	for _, updateCount := range []uint{1, 2, 8, 64, 256} {
		benchmarkName := strconv.FormatUint(uint64(updateCount), 10)
		b.Run(benchmarkName, func(b *testing.B) {
			// Reserve one payload for the snapshot and one for each update.
			relay := stream.NewRelay(updateCount + 1)
			exchangeName := "Benchmark-" + benchmarkName

			ob := &Orderbook{
				exchangeName: exchangeName,
				ob:           make(map[key.PairAsset]*orderbookHolder),
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
				BufferInstance: ob,
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

			if err := ob.LoadSnapshot(book); err != nil {
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
			for range updateCount + 1 {
				<-relay.C
			}

			b.ReportAllocs()
			for b.Loop() {
				if err := ob.LoadSnapshot(book); err != nil {
					b.Fatal(err)
				}
				cache.state = cacheStateQueuing
				if err := manager.applyPendingUpdates(&cache); err != nil {
					b.Fatal(err)
				}
				for range updateCount + 1 {
					<-relay.C
				}
			}
		})
	}
}
