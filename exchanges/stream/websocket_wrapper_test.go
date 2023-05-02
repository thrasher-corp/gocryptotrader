package stream

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var defaultWebsocketWrapperSetup = &WrapperWebsocket{
	AssetTypeWebsockets: map[asset.Item]*Websocket{
		asset.Spot: {
			defaultURL:   "testDefaultURL",
			runningURL:   "wss://testRunningURL",
			connector:    func() error { return nil },
			Subscriber:   func(_ []ChannelSubscription) error { return nil },
			Unsubscriber: func(_ []ChannelSubscription) error { return nil },
			GenerateSubs: func() ([]ChannelSubscription, error) {
				return []ChannelSubscription{
					{Channel: "TestSub"},
					{Channel: "TestSub2"},
					{Channel: "TestSub3"},
					{Channel: "TestSub4"},
				}, nil
			},
		},
	},
}
