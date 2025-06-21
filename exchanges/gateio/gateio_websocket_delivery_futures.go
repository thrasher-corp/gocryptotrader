package gateio

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const (
	// delivery real trading urls
	deliveryRealUSDTTradingURL = "wss://fx-ws.gateio.ws/v4/ws/delivery/usdt"
	deliveryRealBTCTradingURL  = "wss://fx-ws.gateio.ws/v4/ws/delivery/btc"

	// delivery testnet urls
	deliveryTestNetBTCTradingURL  = "wss://fx-ws-testnet.gateio.ws/v4/ws/delivery/btc"  //nolint:unused // Can be used for testing
	deliveryTestNetUSDTTradingURL = "wss://fx-ws-testnet.gateio.ws/v4/ws/delivery/usdt" //nolint:unused // Can be used for testing

	deliveryFuturesUpdateLimit uint64 = 100
)

var defaultDeliveryFuturesSubscriptions = []string{
	futuresTickersChannel,
	futuresTradesChannel,
	futuresOrderbookUpdateChannel,
	futuresCandlesticksChannel,
}

// WsDeliveryFuturesConnect initiates a websocket connection for delivery futures account
func (g *Gateio) WsDeliveryFuturesConnect(ctx context.Context, conn websocket.Connection) error {
	if err := g.CurrencyPairs.IsAssetEnabled(asset.DeliveryFutures); err != nil {
		return err
	}
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}); err != nil {
		return err
	}
	pingMessage, err := json.Marshal(WsInput{
		ID:      conn.GenerateMessageID(false),
		Time:    time.Now().Unix(), // TODO: Func for dynamic time as this will be the same time for every ping message.
		Channel: futuresPingChannel,
	})
	if err != nil {
		return err
	}
	conn.SetupPingHandler(websocketRateLimitNotNeededEPL, websocket.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 5,
		MessageType: gws.PingMessage,
		Message:     pingMessage,
	})
	return nil
}

// GenerateDeliveryFuturesDefaultSubscriptions returns delivery futures default subscriptions params.
// TODO: Update to use the new subscription template system
func (g *Gateio) GenerateDeliveryFuturesDefaultSubscriptions() (subscription.List, error) {
	ctx := context.TODO()
	_, err := g.GetCredentials(ctx)
	if err != nil {
		g.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	channelsToSubscribe := defaultDeliveryFuturesSubscriptions
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		channelsToSubscribe = append(channelsToSubscribe, futuresOrdersChannel, futuresUserTradesChannel, futuresBalancesChannel)
	}

	pairs, err := g.GetEnabledPairs(asset.DeliveryFutures)
	if err != nil {
		if errors.Is(err, asset.ErrNotEnabled) {
			return nil, nil // no enabled pairs, subscriptions require an associated pair.
		}
		return nil, err
	}

	var subscriptions subscription.List
	for i := range channelsToSubscribe {
		for j := range pairs {
			params := make(map[string]any)
			switch channelsToSubscribe[i] {
			case futuresOrderbookChannel:
				params["limit"] = 20
				params["interval"] = "0"
			case futuresCandlesticksChannel:
				params["interval"] = kline.FiveMin
			case futuresOrderbookUpdateChannel:
				params["frequency"] = kline.HundredMilliseconds
				params["level"] = strconv.FormatUint(deliveryFuturesUpdateLimit, 10)
			}
			fPair, err := g.FormatExchangeCurrency(pairs[j], asset.DeliveryFutures)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channelsToSubscribe[i],
				Pairs:   currency.Pairs{fPair.Upper()},
				Params:  params,
				Asset:   asset.DeliveryFutures,
			})
		}
	}
	return subscriptions, nil
}

// DeliveryFuturesSubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) DeliveryFuturesSubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return g.handleSubscription(ctx, conn, subscribeEvent, channelsToUnsubscribe, g.generateDeliveryFuturesPayload)
}

// DeliveryFuturesUnsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) DeliveryFuturesUnsubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return g.handleSubscription(ctx, conn, unsubscribeEvent, channelsToUnsubscribe, g.generateDeliveryFuturesPayload)
}

func (g *Gateio) generateDeliveryFuturesPayload(ctx context.Context, conn websocket.Connection, event string, channelsToSubscribe subscription.List) ([]WsInput, error) {
	if len(channelsToSubscribe) == 0 {
		return nil, errors.New("cannot generate payload, no channels supplied")
	}
	var creds *account.Credentials
	var err error
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		creds, err = g.GetCredentials(ctx)
		if err != nil {
			g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	outbound := make([]WsInput, 0, len(channelsToSubscribe))
	for i := range channelsToSubscribe {
		if len(channelsToSubscribe[i].Pairs) != 1 {
			return nil, subscription.ErrNotSinglePair
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
				sigTemp, err = g.generateWsSignature(creds.Secret, event, channelsToSubscribe[i].Channel, timestamp.Unix())
				if err != nil {
					return nil, err
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
			frequencyString, err = getIntervalString(frequency)
			if err != nil {
				return nil, err
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
				intervalString, err = getIntervalString(interval)
				if err != nil {
					return nil, err
				}
				params = append([]string{intervalString}, params...)
			}
		case futuresOrderbookChannel:
			intervalString, okay := channelsToSubscribe[i].Params["interval"].(string)
			if okay {
				params = append(params, intervalString)
			}
		}
		outbound = append(outbound, WsInput{
			ID:      conn.GenerateMessageID(false),
			Event:   event,
			Channel: channelsToSubscribe[i].Channel,
			Payload: params,
			Auth:    auth,
			Time:    timestamp.Unix(),
		})
	}
	return outbound, nil
}
