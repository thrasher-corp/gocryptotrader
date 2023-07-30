package bybit

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// WsOptionsConnect connects to options a websocket feed
func (by *Bybit) WsOptionsConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.Options) {
		return errWebsocketNotEnabled
	}
	by.Websocket.Conn.SetURL(optionPublic)
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
	go by.wsReadData(asset.Options, by.Websocket.Conn)
	return nil
}

// GenerateOptionsDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateOptionsDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	var channels = []string{chanOrderbook, chanPublicTrade, chanPublicTicker}
	pairs, err := by.GetEnabledPairs(asset.Options)
	if err != nil {
		return nil, err
	}
	for z := range pairs {
		for x := range channels {
			subscriptions = append(subscriptions,
				stream.ChannelSubscription{
					Channel:  channels[x],
					Currency: pairs[z],
					Asset:    asset.Options,
				})
		}
	}
	return subscriptions, nil
}
