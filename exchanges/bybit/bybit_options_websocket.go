package bybit

import (
	"context"
	"net/http"
	"strconv"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// WsOptionsConnect connects to options a websocket feed
func (by *Bybit) WsOptionsConnect() error {
	ctx := context.TODO()
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.Options) {
		return websocket.ErrWebsocketNotEnabled
	}
	by.Websocket.Conn.SetURL(optionPublic)
	var dialer gws.Dialer
	err := by.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}
	pingMessage := PingMessage{Operation: "ping", RequestID: strconv.FormatInt(by.Websocket.Conn.GenerateMessageID(false), 10)}
	pingData, err := json.Marshal(pingMessage)
	if err != nil {
		return err
	}
	by.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     pingData,
		Delay:       bybitWebsocketTimer,
	})

	by.Websocket.Wg.Add(1)
	go by.wsReadData(ctx, asset.Options, by.Websocket.Conn)
	return nil
}

// GenerateOptionsDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateOptionsDefaultSubscriptions() (subscription.List, error) {
	var subscriptions subscription.List
	channels := []string{chanOrderbook, chanPublicTrade, chanPublicTicker}
	pairs, err := by.GetEnabledPairs(asset.Options)
	if err != nil {
		return nil, err
	}
	for z := range pairs {
		for x := range channels {
			subscriptions = append(subscriptions,
				&subscription.Subscription{
					Channel: channels[x],
					Pairs:   currency.Pairs{pairs[z]},
					Asset:   asset.Options,
				})
		}
	}
	return subscriptions, nil
}

// OptionSubscribe sends a subscription message to options public channels.
func (by *Bybit) OptionSubscribe(channelSubscriptions subscription.List) error {
	ctx := context.TODO()
	return by.handleOptionsPayloadSubscription(ctx, "subscribe", channelSubscriptions)
}

// OptionUnsubscribe sends an unsubscription messages through options public channels.
func (by *Bybit) OptionUnsubscribe(channelSubscriptions subscription.List) error {
	ctx := context.TODO()
	return by.handleOptionsPayloadSubscription(ctx, "unsubscribe", channelSubscriptions)
}

func (by *Bybit) handleOptionsPayloadSubscription(ctx context.Context, operation string, channelSubscriptions subscription.List) error {
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
