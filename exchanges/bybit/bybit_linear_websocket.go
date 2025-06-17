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
func (by *Bybit) GenerateLinearDefaultSubscriptions(a asset.Item) (subscription.List, error) {
	pairs, err := by.GetEnabledPairs(a)
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

// LinearSubscribe sends a subscription message to linear public channels.
func (by *Bybit) LinearSubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return by.submitDirectSubscription(ctx, conn, asset.USDTMarginedFutures, "subscribe", channelSubscriptions)
}

// LinearUnsubscribe sends an unsubscription messages through linear public channels.
func (by *Bybit) LinearUnsubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return by.submitDirectSubscription(ctx, conn, asset.USDTMarginedFutures, "unsubscribe", channelSubscriptions)
}
