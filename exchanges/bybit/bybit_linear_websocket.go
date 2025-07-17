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
func (e *Exchange) WsLinearConnect() error {
	ctx := context.TODO()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() || !e.IsAssetWebsocketSupported(asset.LinearContract) {
		return websocket.ErrWebsocketNotEnabled
	}
	e.Websocket.Conn.SetURL(linearPublic)
	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}
	e.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"op": "ping"}`),
		Delay:       bybitWebsocketTimer,
	})

	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx, asset.LinearContract, e.Websocket.Conn)
	if e.IsWebsocketAuthenticationSupported() {
		err = e.WsAuth(ctx)
		if err != nil {
			e.Websocket.DataHandler <- err
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// GenerateLinearDefaultSubscriptions generates default subscription
func (e *Exchange) GenerateLinearDefaultSubscriptions() (subscription.List, error) {
	var subscriptions subscription.List
	channels := []string{chanOrderbook, chanPublicTrade, chanPublicTicker}
	pairs, err := e.GetEnabledPairs(asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	linearPairMap := map[asset.Item]currency.Pairs{
		asset.USDTMarginedFutures: pairs,
	}
	usdcPairs, err := e.GetEnabledPairs(asset.USDCMarginedFutures)
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
func (e *Exchange) LinearSubscribe(channelSubscriptions subscription.List) error {
	ctx := context.TODO()
	return e.handleLinearPayloadSubscription(ctx, "subscribe", channelSubscriptions)
}

// LinearUnsubscribe sends an unsubscription messages through linear public channels.
func (e *Exchange) LinearUnsubscribe(channelSubscriptions subscription.List) error {
	ctx := context.TODO()
	return e.handleLinearPayloadSubscription(ctx, "unsubscribe", channelSubscriptions)
}

func (e *Exchange) handleLinearPayloadSubscription(ctx context.Context, operation string, channelSubscriptions subscription.List) error {
	payloads, err := e.handleSubscriptions(operation, channelSubscriptions)
	if err != nil {
		return err
	}
	for a := range payloads {
		// The options connection does not send the subscription request id back with the subscription notification payload
		// therefore the code doesn't wait for the response to check whether the subscription is successful or not.
		err = e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, payloads[a])
		if err != nil {
			return err
		}
	}
	return nil
}
