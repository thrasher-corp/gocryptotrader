package bybit

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// GenerateOptionsDefaultSubscriptions returns the orderbook and ticker subscriptions for every enabled option pair, and
// a trade subscription for each base coin
func (e *Exchange) GenerateOptionsDefaultSubscriptions() (subscription.List, error) {
	return e.generatePublicDefaultSubscriptions(asset.Options)
}

// OptionsSubscribe subscribes the options connection to subs
func (e *Exchange) OptionsSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	valid, err := e.qualifyPublicSubscriptions(asset.Options, subs)
	return common.AppendError(err, e.sendCommandSubscriptions(ctx, conn, subscribeOperation, valid))
}

// OptionsUnsubscribe unsubscribes the options connection from subs
func (e *Exchange) OptionsUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	valid, err := e.qualifyPublicSubscriptions(asset.Options, subs)
	return common.AppendError(err, e.sendCommandSubscriptions(ctx, conn, unsubscribeOperation, valid))
}
