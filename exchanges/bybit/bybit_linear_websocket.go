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

// WsLinearConnect connects to linear a websocket feed
func (by *Bybit) WsLinearConnect() error {
	ctx := context.TODO()
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.LinearContract) {
		return websocket.ErrWebsocketNotEnabled
	}
	by.Websocket.Conn.SetURL(linearPublic)
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
	go by.wsReadData(ctx, asset.LinearContract, by.Websocket.Conn)
	if by.IsWebsocketAuthenticationSupported() {
		err = by.WsAuth(ctx)
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// GenerateLinearDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateLinearDefaultSubscriptions() (subscription.List, error) {
	var subscriptions subscription.List
	channels := []string{chanOrderbook, chanPublicTrade, chanPublicTicker}
	pairs, err := by.GetEnabledPairs(asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	linearPairMap := map[asset.Item]currency.Pairs{
		asset.USDTMarginedFutures: pairs,
	}
	usdcPairs, err := by.GetEnabledPairs(asset.USDCMarginedFutures)
	if err != nil {
		return nil, err
	}
	linearPairMap[asset.USDCMarginedFutures] = usdcPairs
	pairs = append(pairs, usdcPairs...)
	for a := range linearPairMap {
		for p := range linearPairMap[a] {
			for x := range channels {
				subscriptions = append(subscriptions,
					&subscription.Subscription{
						Channel: channels[x],
						Pairs:   currency.Pairs{pairs[p]},
						Asset:   a,
					})
			}
		}
	}
	return subscriptions, nil
}

// LinearSubscribe sends a subscription message to linear public channels.
func (by *Bybit) LinearSubscribe(channelSubscriptions subscription.List) error {
	ctx := context.TODO()
	return by.handleLinearPayloadSubscription(ctx, "subscribe", channelSubscriptions)
}

// LinearUnsubscribe sends an unsubscription messages through linear public channels.
func (by *Bybit) LinearUnsubscribe(channelSubscriptions subscription.List) error {
	ctx := context.TODO()
	return by.handleLinearPayloadSubscription(ctx, "unsubscribe", channelSubscriptions)
}

func (by *Bybit) handleLinearPayloadSubscription(ctx context.Context, operation string, channelSubscriptions subscription.List) error {
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
