package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/log"
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

// WsDeliveryFuturesConnect initiates a websocket connection for delivery futures account
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
	err = g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	err = g.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  deliveryRealBTCTradingURL,
		RateLimit:            gateioWebsocketRateLimit,
		ResponseCheckTimeout: g.Config.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     g.Config.WebsocketResponseMaxLimit,
		Authenticated:        true,
	})
	if err != nil {
		return err
	}
	err = g.Websocket.AuthConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	g.Websocket.Wg.Add(3)
	go g.wsFunnelConnectionData(g.Websocket.Conn)
	go g.wsFunnelConnectionData(g.Websocket.AuthConn)
	go g.wsReadData()
	if g.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			g.Websocket.GetWebsocketURL())
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
	g.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 5,
		MessageType: websocket.PingMessage,
		Message:     pingMessage,
	})
	return nil
}

// GenerateDeliveryFuturesDefaultSubscriptions returns delivery futures default subscriptions params.
func (g *Gateio) GenerateDeliveryFuturesDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	_, err := g.GetCredentials(context.Background())
	if err != nil {
		g.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	channelsToSubscribe := defaultDeliveryFuturesSubscriptions
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		channelsToSubscribe = append(
			channelsToSubscribe,
			futuresOrdersChannel,
			futuresUserTradesChannel,
			futuresBalancesChannel,
		)
	}
	pairs, err := g.GetAvailablePairs(asset.DeliveryFutures)
	if err != nil {
		return nil, err
	}
	var subscriptions []stream.ChannelSubscription
	for i := range channelsToSubscribe {
		for j := range pairs {
			params := make(map[string]interface{})
			if strings.EqualFold(channelsToSubscribe[i], futuresOrderbookChannel) {
				params["limit"] = 20
				params["interval"] = "0"
			} else if strings.EqualFold(channelsToSubscribe[i], futuresCandlesticksChannel) {
				params["interval"] = kline.FiveMin
			}
			fpair, err := g.FormatExchangeCurrency(pairs[j], asset.DeliveryFutures)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channelsToSubscribe[i],
				Currency: fpair.Upper(),
				Params:   params,
			})
		}
	}
	return subscriptions, nil
}
