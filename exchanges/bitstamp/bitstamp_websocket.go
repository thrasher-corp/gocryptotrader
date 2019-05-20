package bitstamp

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	bitstampWSURL = "wss://ws.bitstamp.net"
)

var tradingPairs map[string]string

// findPairFromChannel extracts the capitalized trading pair from the channel and returns it only if enabled in the config
func (b *Bitstamp) findPairFromChannel(channelName string) (string, error) {
	split := strings.Split(channelName, "_")
	tradingPair := strings.ToUpper(split[len(split)-1])

	for _, enabledPair := range b.EnabledPairs {
		if enabledPair.String() == tradingPair {
			return tradingPair, nil
		}
	}

	return "", errors.New("bistamp_websocket.go error - could not find trading pair")
}

// WsConnect connects to a websocket feed
func (b *Bitstamp) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	tradingPairs = make(map[string]string)
	//
	var dialer websocket.Dialer
	if b.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(b.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(proxy)
	}
	var err error
	b.WebsocketConn, _, err = dialer.Dial(b.Websocket.GetWebsocketURL(), http.Header{})
	if err != nil {
		return fmt.Errorf("%s Unable to connect to Websocket. Error: %s",
			b.Name,
			err)
	}
	if b.Verbose {
		log.Debugf("Successful connection to %v",
			b.Websocket.GetWebsocketURL())
	}
	b.GenerateDefaultSubscriptions()
	go b.WsReadData()
	return nil
}

func (b *Bitstamp) WsReadData() {

}

func (b *Bitstamp) GenerateDefaultSubscriptions() {
	var channels = []string{"live_trades_", "diff_order_book_"}
	enabledCurrencies := b.GetEnabledCurrencies()
	subscriptions := []exchange.WebsocketChannelSubscription{}
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, exchange.WebsocketChannelSubscription{
				Channel:  fmt.Sprintf("%v%v", channels[i], enabledCurrencies[j].Lower().String()),
				Currency: enabledCurrencies[j],
			})
		}
	}
	log.Debugln(subscriptions)
	b.Websocket.SubscribeToChannels(subscriptions)
}

func (b *Bitstamp) Subscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	b.wsRequestMtx.Lock()
	defer b.wsRequestMtx.Unlock()
	if b.Verbose {
		log.Debugf("%v sending message to websocket %v", b.Name, channelToSubscribe)
	}
	return b.wsSend(channelToSubscribe.Channel)
}

func (b *Bitstamp) Unsubscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	b.wsRequestMtx.Lock()
	defer b.wsRequestMtx.Unlock()
	if b.Verbose {
		log.Debugf("%v sending message to websocket %v", b.Name, channelToSubscribe)
	}
	return b.wsSend(channelToSubscribe.Channel)
}

func (b *Bitstamp) wsSend(data interface{}) error {
	b.wsRequestMtx.Lock()
	defer b.wsRequestMtx.Unlock()
	if b.Verbose {
		log.Debugf("%v sending message to websocket %v", b.Name, data)
	}
	return b.WebsocketConn.WriteJSON(data)
}
