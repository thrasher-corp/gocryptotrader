package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

const (
	// delivery real trading urls
	deliveryRealUSDTTradingURL = "wss://fx-ws.gateio.ws/v4/ws/delivery/usdt"
	deliveryRealBTCTradingURL  = "wss://fx-ws.gateio.ws/v4/ws/delivery/btc"

	// delivery testnet urls
	deliveryTestNetBTCTradingURL  = "wss://fx-ws-testnet.gateio.ws/v4/ws/delivery/btc"
	deliveryTestNetUSDTTradingURL = "wss://fx-ws-testnet.gateio.ws/v4/ws/delivery/usdt"
)

var defaultDeliveryFuturesSubscriptions = []string{
	futuresTickersChannel,
	futuresTradesChannel,
	futuresOrderbookChannel,
	futuresCandlesticksChannel,
}

func (g *Gateio) WsDeliveryFuturesConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	futuresAssetType = asset.DeliveryFutures
	var dialer websocket.Dialer
	err := g.Websocket.SetWebsocketURL(deliveryRealUSDTTradingURL, false, true)
	if err != nil {
		return err
	}
	err = g.Websocket.SetWebsocketURL(deliveryRealBTCTradingURL, true, true)
	if err != nil {
		return err
	}
	err = g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	pingMessage, err := json.Marshal(WsInput{
		ID:      g.Websocket.Conn.GenerateMessageID(false),
		Time:    time.Now().Unix(),
		Channel: futuresPingChannel,
	})
	if err != nil {
		return err
	}
	g.Websocket.Wg.Add(1)
	go g.wsReadData()
	go g.WsChannelsMultiplexer.Run()
	g.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 5,
		MessageType: websocket.PingMessage,
		Message:     pingMessage,
	})
	return nil
}

// GenerateDelliveryFuturesDefaultSubscriptions returns delivery futures default subscriptions params.
func (g *Gateio) GenerateDeliveryFuturesDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	_, err := g.GetCredentials(context.Background())
	if err != nil {
		g.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		defaultDeliveryFuturesSubscriptions = append(defaultDeliveryFuturesSubscriptions,
			futuresOrdersChannel,
			futuresUserTradesChannel,
			futuresBalancesChannel,
		)
	}
	var subscriptions []stream.ChannelSubscription
	var pairs []currency.Pair
	pairs, err = g.GetEnabledPairs(asset.DeliveryFutures)
	if err != nil {
		return nil, err
	}
	for i := range defaultDeliveryFuturesSubscriptions {
		for j := range pairs {
			params := make(map[string]interface{})
			if strings.EqualFold(defaultDeliveryFuturesSubscriptions[i], futuresOrderbookChannel) {
				params["limit"] = 20
				params["interval"] = "0"
			} else if strings.EqualFold(defaultDeliveryFuturesSubscriptions[i], futuresCandlesticksChannel) {
				params["interval"] = kline.FiveMin
			}
			fpair, err := g.FormatExchangeCurrency(pairs[j], asset.DeliveryFutures)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  defaultDeliveryFuturesSubscriptions[i],
				Currency: fpair.Upper(),
				Params:   params,
			})
		}
	}
	return subscriptions, nil
}
