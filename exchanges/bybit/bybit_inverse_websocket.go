package bybit

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// WsInverseConnect connects to inverse websocket feed
func (by *Bybit) WsInverseConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.CoinMarginedFutures) {
		return stream.ErrWebsocketNotEnabled
	}
	by.Websocket.Conn.SetURL(inversePublic)
	var dialer websocket.Dialer
	err := by.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	by.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		MessageType: websocket.TextMessage,
		Message:     []byte(`{"op": "ping"}`),
		Delay:       bybitWebsocketTimer,
	})

	by.Websocket.Wg.Add(1)
	go by.wsReadData(asset.CoinMarginedFutures, by.Websocket.Conn)
	return nil
}

// GenerateInverseDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateInverseDefaultSubscriptions() ([]subscription.Subscription, error) {
	var subscriptions []subscription.Subscription
	var channels = []string{chanOrderbook, chanPublicTrade, chanPublicTicker}
	pairs, err := by.GetEnabledPairs(asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	for z := range pairs {
		for x := range channels {
			subscriptions = append(subscriptions,
				subscription.Subscription{
					Channel: channels[x],
					Pair:    pairs[z],
					Asset:   asset.CoinMarginedFutures,
				})
		}
	}
	return subscriptions, nil
}

// InverseSubscribe sends a subscription message to linear public channels.
func (by *Bybit) InverseSubscribe(channelSubscriptions []subscription.Subscription) error {
	return by.handleInversePayloadSubscription("subscribe", channelSubscriptions)
}

// InverseUnsubscribe sends an unsubscription messages through linear public channels.
func (by *Bybit) InverseUnsubscribe(channelSubscriptions []subscription.Subscription) error {
	return by.handleInversePayloadSubscription("unsubscribe", channelSubscriptions)
}

func (by *Bybit) handleInversePayloadSubscription(operation string, channelSubscriptions []subscription.Subscription) error {
	payloads, err := by.handleSubscriptions(asset.CoinMarginedFutures, operation, channelSubscriptions)
	if err != nil {
		return err
	}
	for a := range payloads {
		// The options connection does not send the subscription request id back with the subscription notification payload
		// therefore the code doesn't wait for the response to check whether the subscription is successful or not.
		err = by.Websocket.Conn.SendJSONMessage(payloads[a])
		if err != nil {
			return err
		}
	}
	return nil
}
