package bybit

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// WsLinearConnect connects to linear a websocket feed
func (by *Bybit) WsLinearConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.Linear) {
		return errWebsocketNotEnabled
	}
	by.Websocket.Conn.SetURL(linearPublic)
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
	go by.wsReadData(asset.Linear, by.Websocket.Conn)
	if by.IsWebsocketAuthenticationSupported() {
		err = by.WsAuth(context.TODO())
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// GenerateLinearDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateLinearDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	var channels = []string{chanOrderbook, chanPublicTrade, chanPublicTicker}
	pairs, err := by.GetEnabledPairs(asset.Linear)
	if err != nil {
		return nil, err
	}
	for z := range pairs {
		for x := range channels {
			subscriptions = append(subscriptions,
				stream.ChannelSubscription{
					Channel:  channels[x],
					Currency: pairs[z],
					Asset:    asset.Linear,
				})
		}
	}
	return subscriptions, nil
}
