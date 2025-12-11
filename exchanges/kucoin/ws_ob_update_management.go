package kucoin

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// fetchWSOrderbookSnapshot retrieves a full orderbook snapshot for the specified pair and asset type.
func (e *Exchange) fetchWSOrderbookSnapshot(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	out, err := e.FormatSymbol(p, a)
	if err != nil {
		return nil, err
	}

	var ob *Orderbook
	switch a {
	case asset.Spot:
		ob, err = e.GetOrderbook(ctx, out)
	case asset.Futures:
		ob, err = e.GetFuturesOrderbook(ctx, out)
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if err != nil {
		return nil, err
	}
	return &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		Bids:              ob.Bids,
		Asks:              ob.Asks,
		ValidateOrderbook: e.ValidateOrderbook,
		LastUpdateID:      ob.Sequence,
		LastUpdated:       ob.Time,
		LastPushed:        ob.Time,
	}, nil
}

// From docs: sequenceStart(new) <= sequenceEnd(old) + 1 sequenceEnd(new) > sequenceEnd(old)
// Spot see: https://www.kucoin.com/docs-new/3470221w0
// Futures see: https://www.kucoin.com/docs-new/3470082w0
func checkPendingUpdate(sequenceEndOld, sequenceStartNew int64, update *orderbook.Update) (skip bool, err error) {
	target := sequenceEndOld + 1
	if sequenceStartNew > target {
		return false, buffer.ErrOrderbookSnapshotOutdated
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

func canApplyUpdate(lastUpdateID, firstUpdateID int64) bool {
	return lastUpdateID+1 == firstUpdateID
}
