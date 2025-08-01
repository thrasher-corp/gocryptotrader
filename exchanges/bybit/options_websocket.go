package bybit

import (
	"context"
	"errors"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// GenerateOptionsDefaultSubscriptions generates default subscription
func (e *Exchange) GenerateOptionsDefaultSubscriptions() (subscription.List, error) {
	pairs, err := e.GetEnabledPairs(asset.Options)
	if err != nil {
		if errors.Is(err, asset.ErrNotEnabled) {
			return nil, nil
		}
		return nil, err
	}

	var subscriptions subscription.List
	for z := range pairs {
		for _, channel := range []string{chanOrderbook, chanPublicTrade, chanPublicTicker} {
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channel,
				Pairs:   currency.Pairs{pairs[z]},
				Asset:   asset.Options,
			})
		}
	}
	return subscriptions, nil
}

// OptionsSubscribe sends a websocket message to receive data from the channel
func (e *Exchange) OptionsSubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return e.submitDirectSubscription(ctx, conn, asset.Options, "subscribe", channelSubscriptions)
}

// OptionsUnsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) OptionsUnsubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return e.submitDirectSubscription(ctx, conn, asset.Options, "unsubscribe", channelSubscriptions)
}
