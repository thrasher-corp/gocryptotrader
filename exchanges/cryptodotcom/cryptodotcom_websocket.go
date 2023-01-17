package cryptodotcom

import "github.com/thrasher-corp/gocryptotrader/exchanges/stream"

func (cr *Cryptodotcom) WsConnect() error {
	return nil
}

func (cr *Cryptodotcom) Subscribe(subscriptions []stream.ChannelSubscription) error {
	return nil
}

func (cr *Cryptodotcom) Unsubscribe(subscriptions []stream.ChannelSubscription) error {
	return nil
}

func (cr *Cryptodotcom) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	return nil, nil
}
