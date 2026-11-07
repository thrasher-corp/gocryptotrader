package bybit

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// GenerateLinearDefaultSubscriptions returns the orderbook, trade and ticker subscriptions for every enabled pair of a
// linear asset
func (e *Exchange) GenerateLinearDefaultSubscriptions(a asset.Item) (subscription.List, error) {
	if err := checkLinearAsset(a); err != nil {
		return nil, err
	}
	return e.generatePublicDefaultSubscriptions(a)
}

// LinearSubscribe subscribes the connection of a linear asset to subs
func (e *Exchange) LinearSubscribe(ctx context.Context, conn websocket.Connection, a asset.Item, subs subscription.List) error {
	if err := checkLinearAsset(a); err != nil {
		return err
	}
	valid, err := e.qualifyPublicSubscriptions(a, subs)
	return common.AppendError(err, e.sendSubscriptions(ctx, conn, subscribeOperation, valid))
}

// LinearUnsubscribe unsubscribes the connection of a linear asset from subs
func (e *Exchange) LinearUnsubscribe(ctx context.Context, conn websocket.Connection, a asset.Item, subs subscription.List) error {
	if err := checkLinearAsset(a); err != nil {
		return err
	}
	valid, err := e.qualifyPublicSubscriptions(a, subs)
	return common.AppendError(err, e.sendSubscriptions(ctx, conn, unsubscribeOperation, valid))
}

func checkLinearAsset(a asset.Item) error {
	if a != asset.USDTMarginedFutures && a != asset.USDCMarginedFutures {
		return fmt.Errorf("%q %w for linear subscriptions", a, asset.ErrInvalidAsset)
	}
	return nil
}
