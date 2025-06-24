package bybit

import (
	"context"
	"errors"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// GenerateInverseDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateInverseDefaultSubscriptions() (subscription.List, error) {
	pairs, err := by.GetEnabledPairs(asset.CoinMarginedFutures)
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
				Asset:   asset.CoinMarginedFutures,
			})
		}
	}
	return subscriptions, nil
}

// InverseSubscribe sends a websocket message to receive data from the channel
func (by *Bybit) InverseSubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return by.submitDirectSubscription(ctx, conn, asset.CoinMarginedFutures, "subscribe", channelSubscriptions)
}

// InverseUnsubscribe sends a websocket message to stop receiving data from the channel
func (by *Bybit) InverseUnsubscribe(ctx context.Context, conn websocket.Connection, channelSubscriptions subscription.List) error {
	return by.submitDirectSubscription(ctx, conn, asset.CoinMarginedFutures, "unsubscribe", channelSubscriptions)
}
