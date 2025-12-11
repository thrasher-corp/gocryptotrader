package gateio

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

var (
	errInvalidOrderbookUpdateInterval = errors.New("invalid orderbook update interval")
	spotOrderbookUpdateKey            = subscription.MustChannelKey(subscription.OrderbookChannel)
)

func (e *Exchange) fetchWSOrderbookSnapshot(ctx context.Context, pair currency.Pair, a asset.Item) (*orderbook.Book, error) {
	limit, err := e.extractOrderbookLimit(a)
	if err != nil {
		return nil, err
	}
	return e.fetchOrderbook(ctx, pair, a, limit)
}

// TODO: When subscription config is added for all assets update limits to use sub.Levels
func (e *Exchange) extractOrderbookLimit(a asset.Item) (uint64, error) {
	switch a {
	case asset.Spot:
		sub := e.Websocket.GetSubscription(spotOrderbookUpdateKey)
		if sub == nil {
			return 0, fmt.Errorf("%w for %q", subscription.ErrNotFound, spotOrderbookUpdateKey)
		}
		// There is no way to set levels when we subscribe for this specific channel
		// Extract limit from interval e.g. 20ms == 20 limit book and 100ms == 100 limit book.
		lim := uint64(sub.Interval.Duration().Milliseconds()) //nolint:gosec // No overflow risk
		if lim != 20 && lim != 100 {
			return 0, fmt.Errorf("%w: %d. Valid limits are 20 and 100", errInvalidOrderbookUpdateInterval, lim)
		}
		return lim, nil
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		return futuresOrderbookUpdateLimit, nil
	case asset.DeliveryFutures:
		return deliveryFuturesUpdateLimit, nil
	case asset.Options:
		return optionOrderbookUpdateLimit, nil
	default:
		return 0, fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
}

func checkPendingUpdate(lastUpdateID int64, firstUpdateID int64, update *orderbook.Update) (skip bool, err error) {
	nextUpdateID := lastUpdateID + 1 // From docs: `baseId+1`

	// From docs: Dump all notifications which satisfy `u` < `baseId+1`
	if update.UpdateID < nextUpdateID {
		return true, nil
	}

	// From docs: `baseID+1` < first notification `U` current base order book falls behind notifications
	if nextUpdateID < firstUpdateID {
		return false, buffer.ErrOrderbookSnapshotOutdated
	}

	return false, nil
}

func canApplyUpdate(lastUpdateID int64, firstUpdateID int64) bool {
	return lastUpdateID+1 == firstUpdateID
}
