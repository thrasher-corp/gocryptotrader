package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
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

var fetchedFuturesCurrencyPairSnapshotOrderbook = make(map[string]bool)

// WsDeliveryFuturesConnect initiates a websocket connection for delivery futures account
func (g *Gateio) WsDeliveryFuturesConnect() error {
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
	go g.wsReadDeliveryFuturesData()
	go g.wsFunnelDeliveryFuturesConnectionData(g.Websocket.Conn)
	go g.wsFunnelDeliveryFuturesConnectionData(g.Websocket.AuthConn)
	if g.Verbose {
		log.Debugf(log.ExchangeSys, "successful connection to %v\n",
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
	g.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 5,
		MessageType: websocket.PingMessage,
		Message:     pingMessage,
	})
	return nil
}

// wsReadDeliveryFuturesData read coming messages thought the websocket connection and pass the data to wsHandleFuturesData for further process.
func (g *Gateio) wsReadDeliveryFuturesData() {
	defer g.Websocket.Wg.Done()
	for {
		select {
		case <-g.Websocket.ShutdownC:
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
			})
		}
	}
	return subscriptions, nil
}

// DeliveryFuturesSubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) DeliveryFuturesSubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return g.handleDeliveryFuturesSubscription("subscribe", channelsToUnsubscribe)
}

// DeliveryFuturesUnsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) DeliveryFuturesUnsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return g.handleDeliveryFuturesSubscription("unsubscribe", channelsToUnsubscribe)
}

// handleDeliveryFuturesSubscription sends a websocket message to receive data from the channel
func (g *Gateio) handleDeliveryFuturesSubscription(event string, channelsToSubscribe []stream.ChannelSubscription) error {
	payloads, err := g.generateDeliveryFuturesPayload(event, channelsToSubscribe)
	if err != nil {
		return err
	}
	var errs error
	var respByte []byte
	// con represents the websocket connection. 0 - for usdt settle and 1 - for btc settle connections.
	for con, val := range payloads {
		for k := range val {
			if con == 0 {
				respByte, err = g.Websocket.Conn.SendMessageReturnResponse(val[k].ID, val[k])
			} else {
				respByte, err = g.Websocket.AuthConn.SendMessageReturnResponse(val[k].ID, val[k])
			}
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			var resp WsEventResponse
			if err = json.Unmarshal(respByte, &resp); err != nil {
				errs = common.AppendError(errs, err)
			} else {
				if resp.Error != nil && resp.Error.Code != 0 {
					errs = common.AppendError(errs, fmt.Errorf("error while %s to channel %s error code: %d message: %s", val[k].Event, val[k].Channel, resp.Error.Code, resp.Error.Message))
					continue
				}
				g.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[k])
			}
		}
	}
	return errs
}

func (g *Gateio) generateDeliveryFuturesPayload(event string, channelsToSubscribe []stream.ChannelSubscription) ([2][]WsInput, error) {
	if len(channelsToSubscribe) == 0 {
		return [2][]WsInput{}, errors.New("cannot generate payload, no channels supplied")
	}
	var creds *account.Credentials
	var err error
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		creds, err = g.GetCredentials(context.TODO())
		if err != nil {
			g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	payloads := [2][]WsInput{}
	for i := range channelsToSubscribe {
		var auth *WsAuthInput
		timestamp := time.Now()
		var params []string
		params = []string{channelsToSubscribe[i].Currency.String()}
		if g.Websocket.CanUseAuthenticatedEndpoints() {
			switch channelsToSubscribe[i].Channel {
			case futuresOrdersChannel, futuresUserTradesChannel,
				futuresLiquidatesChannel, futuresAutoDeleveragesChannel,
				futuresAutoPositionCloseChannel, futuresBalancesChannel,
				futuresReduceRiskLimitsChannel, futuresPositionsChannel,
				futuresAutoOrdersChannel:
				value, ok := channelsToSubscribe[i].Params["user"].(string)
				if ok {
					params = append(
						[]string{value},
						params...)
				}
				var sigTemp string
				sigTemp, err = g.generateWsSignature(creds.Secret, event, channelsToSubscribe[i].Channel, timestamp)
				if err != nil {
					return [2][]WsInput{}, err
				}
				auth = &WsAuthInput{
					Method: "api_key",
					Key:    creds.Key,
					Sign:   sigTemp,
				}
			}
		}
		frequency, okay := channelsToSubscribe[i].Params["frequency"].(kline.Interval)
		if okay {
			var frequencyString string
			frequencyString, err = g.GetIntervalString(frequency)
			if err != nil {
				return payloads, err
			}
			params = append(params, frequencyString)
		}
		levelString, okay := channelsToSubscribe[i].Params["level"].(string)
		if okay {
			params = append(params, levelString)
		}
		limit, okay := channelsToSubscribe[i].Params["limit"].(int)
		if okay {
			params = append(params, strconv.Itoa(limit))
		}
		accuracy, okay := channelsToSubscribe[i].Params["accuracy"].(string)
		if okay {
			params = append(params, accuracy)
		}
		switch channelsToSubscribe[i].Channel {
		case futuresCandlesticksChannel:
			interval, okay := channelsToSubscribe[i].Params["interval"].(kline.Interval)
			if okay {
				var intervalString string
				intervalString, err = g.GetIntervalString(interval)
				if err != nil {
					return payloads, err
				}
				params = append([]string{intervalString}, params...)
			}
		case futuresOrderbookChannel:
			intervalString, okay := channelsToSubscribe[i].Params["interval"].(string)
			if okay {
				params = append(params, intervalString)
			}
		}
		if strings.HasPrefix(channelsToSubscribe[i].Currency.Quote.Upper().String(), "USDT") {
			payloads[0] = append(payloads[0], WsInput{
				ID:      g.Websocket.Conn.GenerateMessageID(false),
				Event:   event,
				Channel: channelsToSubscribe[i].Channel,
				Payload: params,
				Auth:    auth,
				Time:    timestamp.Unix(),
			})
		} else {
			payloads[1] = append(payloads[1], WsInput{
				ID:      g.Websocket.Conn.GenerateMessageID(false),
				Event:   event,
				Channel: channelsToSubscribe[i].Channel,
				Payload: params,
				Auth:    auth,
				Time:    timestamp.Unix(),
			})
		}
	}
	return payloads, nil
}
