package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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

// responseDeliveryFuturesStream a channel thought which the data coming from the two websocket connection will go through.
var responseDeliveryFuturesStream = make(chan stream.Response)

var fetchedFuturesCurrencyPairSnapshotOrderbook map[string]bool

// WsDeliveryFuturesConnect initiates a websocket connection for delivery futures account
func (g *Gateio) WsDeliveryFuturesConnect() error {
	fetchedFuturesCurrencyPairSnapshotOrderbook = make(map[string]bool)
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	err := g.CurrencyPairs.IsAssetEnabled(asset.DeliveryFutures)
	if err != nil {
		return err
	}
	var dialer websocket.Dialer
	err = g.Websocket.SetWebsocketURL(deliveryRealUSDTTradingURL, false, true)
	if err != nil {
		return err
	}
	err = g.Websocket.AssetTypeWebsockets[asset.DeliveryFutures].Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	err = g.Websocket.AssetTypeWebsockets[asset.DeliveryFutures].SetupNewConnection(stream.ConnectionSetup{
		URL:                  deliveryRealBTCTradingURL,
		RateLimit:            gateioWebsocketRateLimit,
		ResponseCheckTimeout: g.Config.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     g.Config.WebsocketResponseMaxLimit,
		Authenticated:        true,
	})
	if err != nil {
		return err
	}
	err = g.Websocket.AssetTypeWebsockets[asset.DeliveryFutures].AuthConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	g.Websocket.Wg.Add(3)
	go g.wsReadDeliveryFuturesData()
	go g.wsFunnelDeliveryFuturesConnectionData(g.Websocket.AssetTypeWebsockets[asset.DeliveryFutures].Conn)
	go g.wsFunnelDeliveryFuturesConnectionData(g.Websocket.AssetTypeWebsockets[asset.DeliveryFutures].AuthConn)
	if g.Verbose {
		log.Debugf(log.ExchangeSys, "successful connection to %v\n",
			g.Websocket.GetWebsocketURL())
	}
	pingMessage, err := json.Marshal(WsInput{
		ID:      g.Websocket.AssetTypeWebsockets[asset.DeliveryFutures].Conn.GenerateMessageID(false),
		Time:    time.Now().Unix(),
		Channel: futuresPingChannel,
	})
	if err != nil {
		return err
	}
	g.Websocket.AssetTypeWebsockets[asset.DeliveryFutures].Conn.SetupPingHandler(stream.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 5,
		MessageType: websocket.PingMessage,
		Message:     pingMessage,
	})
	return nil
}

// wsReadDeliveryFuturesData read coming messages thought the websocket connection and pass the data to wsHandleData for further process.
func (g *Gateio) wsReadDeliveryFuturesData() {
	defer g.Websocket.Wg.Done()
	for {
		select {
		case <-g.Websocket.AssetTypeWebsockets[asset.DeliveryFutures].ShutdownC:
			select {
			case resp := <-responseDeliveryFuturesStream:
				err := g.wsHandleFuturesData(resp.Raw, asset.DeliveryFutures)
				if err != nil {
					select {
					case g.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr, "%s websocket handle data error: %v", g.Name, err)
					}
				}
			default:
			}
			return
		case resp := <-responseDeliveryFuturesStream:
			err := g.wsHandleFuturesData(resp.Raw, asset.DeliveryFutures)
			if err != nil {
				g.Websocket.DataHandler <- err
			}
		}
	}
}

// wsFunnelDeliveryFuturesConnectionData receives data from multiple connection and pass the data
// to wsRead through a channel responseStream
func (g *Gateio) wsFunnelDeliveryFuturesConnectionData(ws stream.Connection) {
	defer g.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		responseDeliveryFuturesStream <- stream.Response{Raw: resp.Raw}
	}
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
			switch channelsToSubscribe[i] {
			case futuresOrderbookChannel:
				params["limit"] = 20
				params["interval"] = "0"
			case futuresCandlesticksChannel:
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
				Asset:    asset.DeliveryFutures,
			})
		}
	}
	return subscriptions, nil
}
