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
func (by *Bybit) GenerateOptionsDefaultSubscriptions() (subscription.List, error) {
	pairs, err := by.GetEnabledPairs(asset.Options)
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

// OptionSubscribe sends a subscription message to options public channels.
func (by *Bybit) OptionSubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return by.submitSubscriptionNonTemplate(ctx, conn, asset.Options, "subscribe", channelSubscriptions)
}

// OptionUnsubscribe sends an unsubscription messages through options public channels.
func (by *Bybit) OptionUnsubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return by.submitSubscriptionNonTemplate(ctx, conn, asset.Options, "unsubscribe", channelSubscriptions)
}
