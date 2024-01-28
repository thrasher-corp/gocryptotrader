package bybit

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// WsOptionsConnect connects to options a websocket feed
func (by *Bybit) WsOptionsConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.Options) {
		return errWebsocketNotEnabled
	}
	optionsWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Options)
	if err != nil {
		return err
	}
	optionsWebsocket.Conn.SetURL(optionPublic)
	var dialer websocket.Dialer
	err = optionsWebsocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	pingMessage := PingMessage{Operation: "ping", RequestID: strconv.FormatInt(optionsWebsocket.Conn.GenerateMessageID(false), 10)}
	pingData, err := json.Marshal(pingMessage)
	if err != nil {
		return err
	}
	optionsWebsocket.Conn.SetupPingHandler(stream.PingHandler{
		MessageType: websocket.TextMessage,
		Message:     pingData,
		Delay:       bybitWebsocketTimer,
	})

	by.Websocket.Wg.Add(1)
	go by.wsReadData(asset.Options, optionsWebsocket.Conn)
	return nil
}

// GenerateOptionsDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateOptionsDefaultSubscriptions() ([]subscription.Subscription, error) {
	var subscriptions []subscription.Subscription
	var channels = []string{chanOrderbook, chanPublicTrade, chanPublicTicker}
	pairs, err := by.GetEnabledPairs(asset.Options)
	if err != nil {
		return nil, err
	}
	for z := range pairs {
		for x := range channels {
			subscriptions = append(subscriptions,
				subscription.Subscription{
					Channel: channels[x],
					Pair:    pairs[z],
					Asset:   asset.Options,
				})
		}
	}
	return subscriptions, nil
}

// OptionSubscribe sends a subscription message to options public channels.
func (by *Bybit) OptionSubscribe(channelSubscriptions []subscription.Subscription) error {
	return by.handleOptionsPayloadSubscription("subscribe", channelSubscriptions)
}

// OptionUnsubscribe sends an unsubscription messages through options public channels.
func (by *Bybit) OptionUnsubscribe(channelSubscriptions []subscription.Subscription) error {
	return by.handleOptionsPayloadSubscription("unsubscribe", channelSubscriptions)
}

func (by *Bybit) handleOptionsPayloadSubscription(operation string, channelSubscriptions []subscription.Subscription) error {
	optionsWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Options)
	if err != nil {
		return err
	}
	payloads, err := by.handleSubscriptions(asset.Options, operation, channelSubscriptions)
	if err != nil {
		return err
	}
	for a := range payloads {
		// The options connection does not send the subscription request id back with the subscription notification payload
		// therefore the code doesn't wait for the response to check whether the subscription is successful or not.
		err = optionsWebsocket.Conn.SendJSONMessage(payloads[a])
		if err != nil {
			return err
		}
	}
	return nil
}
