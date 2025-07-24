package bybit

import (
	"context"
	"errors"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// GenerateLinearDefaultSubscriptions generates default subscription
func (e *Exchange) GenerateLinearDefaultSubscriptions(a asset.Item) (subscription.List, error) {
	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		if errors.Is(err, asset.ErrNotEnabled) {
			return nil, nil
		}
		return nil, err
	}

	var subscriptions subscription.List
	for _, pair := range pairs {
		for _, channel := range []string{chanOrderbook, chanPublicTrade, chanPublicTicker} {
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channel,
				Pairs:   currency.Pairs{pair},
				Asset:   a,
			})
		}
	}
	return subscriptions, nil
}

// LinearSubscribe sends a websocket message to receive data from the channel
func (e *Exchange) LinearSubscribe(ctx context.Context, conn websocket.Connection, a asset.Item, channelSubscriptions subscription.List) error {
	return e.submitDirectSubscription(ctx, conn, a, "subscribe", channelSubscriptions)
}

// LinearUnsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) LinearUnsubscribe(ctx context.Context, conn websocket.Connection, a asset.Item, channelSubscriptions subscription.List) error {
	return e.submitDirectSubscription(ctx, conn, a, "unsubscribe", channelSubscriptions)
}
