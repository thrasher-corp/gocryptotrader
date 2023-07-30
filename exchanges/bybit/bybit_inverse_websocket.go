package bybit

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// WsInverseConnect connects to inverse websocket feed
func (by *Bybit) WsInverseConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.Inverse) {
		return errWebsocketNotEnabled
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
	go by.wsReadData(asset.Inverse, by.Websocket.Conn)
	return nil
}

// GenerateInverseDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateInverseDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	var channels = []string{chanOrderbook, chanPublicTrade, chanPublicTicker}
	pairs, err := by.GetEnabledPairs(asset.Inverse)
	if err != nil {
		return nil, err
	}
	for z := range pairs {
		for x := range channels {
			subscriptions = append(subscriptions,
				stream.ChannelSubscription{
					Channel:  channels[x],
					Currency: pairs[z],
					Asset:    asset.Inverse,
				})
		}
	}
	return subscriptions, nil
}
