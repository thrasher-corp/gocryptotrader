package kucoin

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// newOrderbookUpdateManager returns the manager that syncs the Orderbook - Increment topics' books with their
// snapshots, keeping them in the store of the exchange's current websocket
func (e *Exchange) newOrderbookUpdateManager() *orderbookmanager.UpdateManager {
	return orderbookmanager.NewUpdateManager(&orderbookmanager.UpdateManagerParams{
		FetchDelay:         orderbookmanager.DefaultWSOrderbookUpdateTimeDelay,
		FetchDeadline:      orderbookmanager.DefaultWSOrderbookUpdateDeadline,
		FetchOrderbook:     e.fetchWSOrderbookSnapshot,
		CheckPendingUpdate: checkPendingUpdate,
		Orderbook:          &e.Websocket.Orderbook,
	})
}

// fetchWSOrderbookSnapshot retrieves the book the Orderbook - Increment topics calibrate against, as the docs name it:
// Get Full OrderBook of the spot or futures market
func (e *Exchange) fetchWSOrderbookSnapshot(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	out, err := e.FormatSymbol(p, a)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	switch a {
	case asset.Spot, asset.Margin:
		ob, err := e.GetOrderbook(ctx, out)
		if err != nil {
			return nil, err
		}
		book.Bids, book.Asks = ob.Bids.Levels(), ob.Asks.Levels()
		book.LastUpdateID = ob.Sequence.Int64()
		book.LastUpdated, book.LastPushed = ob.Time.Time(), ob.Time.Time()
	case asset.Futures:
		ob, err := e.GetFuturesOrderbook(ctx, out)
		if err != nil {
			return nil, err
		}
		book.Bids, book.Asks = ob.Bids.Levels(), ob.Asks.Levels()
		book.LastUpdateID = int64(ob.Sequence) //nolint:gosec // KuCoin's sequences stay far below the int64 limit
		book.LastUpdated, book.LastPushed = ob.Timestamp.Time(), ob.Timestamp.Time()
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	return book, nil
}

// From docs: sequenceStart(new) <= sequenceEnd(old) + 1 sequenceEnd(new) > sequenceEnd(old)
// Spot see: https://www.kucoin.com/docs-new/3470068w0
// Futures see: https://www.kucoin.com/docs-new/3470082w0
func checkPendingUpdate(sequenceEndOld, sequenceStartNew int64, update *orderbook.Update) (skip bool, err error) {
	target := sequenceEndOld + 1
	if sequenceStartNew > target {
		return false, orderbookmanager.ErrOrderbookSnapshotOutdated
	}

	if update.UpdateID < target {
		return true, nil
	}

	// Trim levels that are not required
	bids := make(orderbook.Levels, 0, len(update.Bids))
	for i := range update.Bids {
		if update.Bids[i].ID >= target {
			bids = append(bids, update.Bids[i])
		}
	}
	update.Bids = bids
	asks := make(orderbook.Levels, 0, len(update.Asks))
	for i := range update.Asks {
		if update.Asks[i].ID >= target {
			asks = append(asks, update.Asks[i])
		}
	}
	update.Asks = asks
	return false, nil
}
