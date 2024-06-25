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
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// delivery real trading urls
	deliveryRealUSDTTradingURL = "wss://fx-ws.gateio.ws/v4/ws/delivery/usdt"
	deliveryRealBTCTradingURL  = "wss://fx-ws.gateio.ws/v4/ws/delivery/btc"
)

var defaultDeliveryFuturesSubscriptions = []string{
	futuresTickersChannel,
	futuresTradesChannel,
	futuresOrderbookChannel,
	futuresCandlesticksChannel,
}

var fetchedFuturesCurrencyPairSnapshotOrderbook = make(map[string]bool)

// WsDeliveryFuturesConnect initiates a websocket connection for delivery futures account
func (g *Gateio) WsDeliveryFuturesConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() || !g.IsAssetWebsocketSupported(asset.DeliveryFutures) {
		return fmt.Errorf("%w for asset type %s", stream.ErrWebsocketNotEnabled, asset.DeliveryFutures)
	}
	deliveryFuturesWebsocket, err := g.Websocket.GetAssetWebsocket(asset.DeliveryFutures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.DeliveryFutures)
	}
	err = g.CurrencyPairs.IsAssetEnabled(asset.DeliveryFutures)
	if err != nil {
		return err
	}
	var dialer websocket.Dialer
	err = deliveryFuturesWebsocket.SetWebsocketURL(deliveryRealUSDTTradingURL, false, true)
	if err != nil {
		return err
	}
	err = deliveryFuturesWebsocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	err = deliveryFuturesWebsocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  deliveryRealBTCTradingURL,
		RateLimit:            gateioWebsocketRateLimit,
		ResponseCheckTimeout: g.Config.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     g.Config.WebsocketResponseMaxLimit,
		Authenticated:        true,
	})
	if err != nil {
		return err
	}
	err = deliveryFuturesWebsocket.AuthConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	go g.wsReadDeliveryFuturesData(deliveryFuturesWebsocket.Conn)
	go g.wsReadDeliveryFuturesData(deliveryFuturesWebsocket.AuthConn)
	if g.Verbose {
		log.Debugf(log.ExchangeSys, "successful connection to %v\n",
			g.Websocket.GetWebsocketURL())
	}
	pingMessage, err := json.Marshal(WsInput{
		ID:      deliveryFuturesWebsocket.Conn.GenerateMessageID(false),
		Time:    time.Now().Unix(),
		Channel: futuresPingChannel,
	})
	if err != nil {
		return err
	}
	deliveryFuturesWebsocket.Conn.SetupPingHandler(stream.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 5,
		MessageType: websocket.PingMessage,
		Message:     pingMessage,
	})
	return nil
}

// wsReadDeliveryFuturesData read coming messages thought the websocket connection and pass the data to wsHandleFuturesData for further process.
func (g *Gateio) wsReadDeliveryFuturesData(ws stream.Connection) {
	deliveryFuturesWebsocket, err := g.Websocket.GetAssetWebsocket(asset.DeliveryFutures)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v asset type: %v", err, asset.DeliveryFutures)
	}
	deliveryFuturesWebsocket.Wg.Add(1)
	defer deliveryFuturesWebsocket.Wg.Done()
	for {
		select {
		case <-deliveryFuturesWebsocket.ShutdownC:
			return
		default:
			resp := ws.ReadMessage()
			if resp.Raw == nil {
				return
			}
			err := g.wsHandleFuturesData(resp.Raw, asset.DeliveryFutures)
			if err != nil {
				g.Websocket.DataHandler <- err
			}
		}
	}
}

// GenerateDeliveryFuturesDefaultSubscriptions returns delivery futures default subscriptions params.
func (g *Gateio) GenerateDeliveryFuturesDefaultSubscriptions() (subscription.List, error) {
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
	var subscriptions subscription.List
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
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channelsToSubscribe[i],
				Pairs:   currency.Pairs{fpair.Upper()},
				Params:  params,
				Asset:   asset.DeliveryFutures,
			})
		}
	}
	return subscriptions, nil
}

// DeliveryFuturesSubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) DeliveryFuturesSubscribe(channelsToUnsubscribe subscription.List) error {
	return g.handleDeliveryFuturesSubscription("subscribe", channelsToUnsubscribe)
}

// DeliveryFuturesUnsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) DeliveryFuturesUnsubscribe(channelsToUnsubscribe subscription.List) error {
	return g.handleDeliveryFuturesSubscription("unsubscribe", channelsToUnsubscribe)
}

// handleDeliveryFuturesSubscription sends a websocket message to receive data from the channel
func (g *Gateio) handleDeliveryFuturesSubscription(event string, channelsToSubscribe subscription.List) error {
	deliveryFuturesWebsocket, err := g.Websocket.GetAssetWebsocket(asset.DeliveryFutures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.DeliveryFutures)
	}
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
				respByte, err = deliveryFuturesWebsocket.Conn.SendMessageReturnResponse(val[k].ID, val[k])
			} else {
				respByte, err = deliveryFuturesWebsocket.AuthConn.SendMessageReturnResponse(val[k].ID, val[k])
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
				if err = deliveryFuturesWebsocket.AddSuccessfulSubscriptions(channelsToSubscribe[k]); err != nil {
					errs = common.AppendError(errs, err)
				}
			}
		}
	}
	return errs
}

func (g *Gateio) generateDeliveryFuturesPayload(event string, channelsToSubscribe subscription.List) ([2][]WsInput, error) {
	payloads := [2][]WsInput{}
	if len(channelsToSubscribe) == 0 {
		return payloads, errors.New("cannot generate payload, no channels supplied")
	}
	deliveryFuturesWebsocket, err := g.Websocket.GetAssetWebsocket(asset.DeliveryFutures)
	if err != nil {
		return [2][]WsInput{}, fmt.Errorf("%w asset type: %v", err, asset.DeliveryFutures)
	}
	var creds *account.Credentials
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		creds, err = g.GetCredentials(context.TODO())
		if err != nil {
			g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	for i := range channelsToSubscribe {
		if len(channelsToSubscribe[i].Pairs) != 1 {
			return payloads, subscription.ErrNotSinglePair
		}
		var auth *WsAuthInput
		timestamp := time.Now()
		var params []string
		params = []string{channelsToSubscribe[i].Pairs[0].String()}
		if g.Websocket.CanUseAuthenticatedEndpoints() {
			switch channelsToSubscribe[i].Channel {
			case futuresOrdersChannel, futuresUserTradesChannel,
				futuresLiquidatesChannel, futuresAutoDeleveragesChannel,
				futuresAutoPositionCloseChannel, futuresBalancesChannel,
				futuresReduceRiskLimitsChannel, futuresPositionsChannel,
				futuresAutoOrdersChannel:
				value, ok := channelsToSubscribe[i].Params["user"].(string)
				if ok {
					params = append([]string{value}, params...)
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
		if strings.HasPrefix(channelsToSubscribe[i].Pairs[0].Quote.Upper().String(), "USDT") {
			payloads[0] = append(payloads[0], WsInput{
				ID:      deliveryFuturesWebsocket.Conn.GenerateMessageID(false),
				Event:   event,
				Channel: channelsToSubscribe[i].Channel,
				Payload: params,
				Auth:    auth,
				Time:    timestamp.Unix(),
			})
		} else {
			payloads[1] = append(payloads[1], WsInput{
				ID:      deliveryFuturesWebsocket.Conn.GenerateMessageID(false),
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
