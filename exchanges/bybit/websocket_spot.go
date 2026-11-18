package bybit

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// SpotSubscribe subscribes the spot connection to subs
func (e *Exchange) SpotSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.handleSpotSubscription(ctx, conn, subscribeOperation, subs)
}

// SpotUnsubscribe unsubscribes the spot connection from subs
func (e *Exchange) SpotUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.handleSpotSubscription(ctx, conn, unsubscribeOperation, subs)
}

func (e *Exchange) handleSpotSubscription(ctx context.Context, conn websocket.Connection, operation string, subs subscription.List) error {
	subs, err := subs.ExpandTemplates(e)
	if err != nil {
		return err
	}
	return e.sendSubscriptions(ctx, conn, operation, subs)
}
