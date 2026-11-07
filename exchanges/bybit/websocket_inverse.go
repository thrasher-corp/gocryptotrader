package bybit

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// GenerateInverseDefaultSubscriptions returns the orderbook, trade and ticker subscriptions for every enabled inverse
// pair
func (e *Exchange) GenerateInverseDefaultSubscriptions() (subscription.List, error) {
	return e.generatePublicDefaultSubscriptions(asset.CoinMarginedFutures)
}

// InverseSubscribe subscribes the inverse connection to subs
func (e *Exchange) InverseSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	valid, err := e.qualifyPublicSubscriptions(asset.CoinMarginedFutures, subs)
	return common.AppendError(err, e.sendSubscriptions(ctx, conn, subscribeOperation, valid))
}

// InverseUnsubscribe unsubscribes the inverse connection from subs
func (e *Exchange) InverseUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	valid, err := e.qualifyPublicSubscriptions(asset.CoinMarginedFutures, subs)
	return common.AppendError(err, e.sendSubscriptions(ctx, conn, unsubscribeOperation, valid))
}
