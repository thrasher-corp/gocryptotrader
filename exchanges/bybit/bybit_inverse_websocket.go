package bybit

import (
	"context"
	"net/http"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// WsInverseConnect connects to inverse websocket feed
func (by *Bybit) WsInverseConnect() error {
	ctx := context.TODO()
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.CoinMarginedFutures) {
		return websocket.ErrWebsocketNotEnabled
	}
	by.Websocket.Conn.SetURL(inversePublic)
	var dialer gws.Dialer
	err := by.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}
	by.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"op": "ping"}`),
		Delay:       bybitWebsocketTimer,
	})

	by.Websocket.Wg.Add(1)
	go by.wsReadData(ctx, asset.CoinMarginedFutures, by.Websocket.Conn)
	return nil
}

// GenerateInverseDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateInverseDefaultSubscriptions() (subscription.List, error) {
	var subscriptions subscription.List
	channels := []string{chanOrderbook, chanPublicTrade, chanPublicTicker}
	pairs, err := by.GetEnabledPairs(asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	for z := range pairs {
		for x := range channels {
			subscriptions = append(subscriptions,
				&subscription.Subscription{
					Channel: channels[x],
					Pairs:   currency.Pairs{pairs[z]},
					Asset:   asset.CoinMarginedFutures,
				})
		}
	}
	return subscriptions, nil
}

// InverseSubscribe sends a subscription message to linear public channels.
func (by *Bybit) InverseSubscribe(channelSubscriptions subscription.List) error {
	ctx := context.TODO()
	return by.handleInversePayloadSubscription(ctx, "subscribe", channelSubscriptions)
}

// InverseUnsubscribe sends an unsubscription messages through linear public channels.
func (by *Bybit) InverseUnsubscribe(channelSubscriptions subscription.List) error {
	ctx := context.TODO()
	return by.handleInversePayloadSubscription(ctx, "unsubscribe", channelSubscriptions)
}

func (by *Bybit) handleInversePayloadSubscription(ctx context.Context, operation string, channelSubscriptions subscription.List) error {
	payloads, err := by.handleSubscriptions(operation, channelSubscriptions)
	if err != nil {
		return err
	}
	for a := range payloads {
		// The options connection does not send the subscription request id back with the subscription notification payload
		// therefore the code doesn't wait for the response to check whether the subscription is successful or not.
		err = by.Websocket.Conn.SendJSONMessage(ctx, request.Unset, payloads[a])
		if err != nil {
			return err
		}
	}
	return nil
}
