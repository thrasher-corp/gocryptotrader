package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	binanceUFuturesWebsocketURL     = "wss://fstream.binance.com"
	binanceUFuturesAuthWebsocketURL = "wss://fstream-auth.binance.com"

	// channels

	assetIndexAllChan   = "!assetIndex@arr"
	contractInfoAllChan = "!contractInfo"
	forceOrderAllChan   = "!forceOrder@arr"
	bookTickerAllChan   = "!bookTicker"
	tickerAllChan       = "!ticker@arr"
	miniTickerAllChan   = "!miniTicker@arr"
	markPriceAllChan    = "!markPrice@arr" // !markPrice@arr  !markPrice@arr@1s

	aggTradeChan       = "@aggTrade" // <symbol>@aggTrade
	depthChan          = "@depth"
	markPriceChan      = "@markPrice" // <symbol>@markPrice <symbol>@markPrice@1s
	tickerChan         = "@ticker"    // <symbol>@ticker
	klineChan          = "@kline"     // <symbol>@kline_<interval>
	miniTickerChan     = "@miniTicker"
	bookTickersChan    = "@bookTickers"
	forceOrderChan     = "@forceOrder"
	compositeIndexChan = "@compositeIndex"
	assetIndexChan     = "@assetIndex"
)

var defaultSubscriptions = []string{
	depthChan,
	klineChan,
	tickerChan,
	aggTradeChan,
}

// getKlineIntervalString returns a string representation of the kline interval.
func getKlineIntervalString(interval kline.Interval) string {
	klineMap := map[kline.Interval]string{
		kline.OneMin: "1m", kline.ThreeMin: "3m", kline.FiveMin: "5m", kline.FifteenMin: "15m", kline.ThirtyMin: "30m",
		kline.OneHour: "1h", kline.TwoHour: "2h", kline.FourHour: "4h", kline.SixHour: "6h", kline.EightHour: "8h", kline.TwelveHour: "12h",
		kline.OneDay: "1d", kline.ThreeDay: "3d", kline.OneWeek: "1w", kline.OneMonth: "1M",
	}
	intervalString, okay := klineMap[interval]
	if !okay {
		return ""
	}
	return intervalString
}

// WsUFuturesConnect initiates a websocket connection
func (b *Binance) WsUFuturesConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	dialer.HandshakeTimeout = b.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	var err error
	wsURL := binanceUFuturesWebsocketURL + "/stream"
	err = b.Websocket.SetWebsocketURL(wsURL, false, false)
	if err != nil {
		return err
	}
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		listenKey, err = b.GetWsAuthStreamKey(context.TODO())
		if err != nil {
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys,
				"%v unable to connect to authenticated Websocket. Error: %s",
				b.Name,
				err)
		} else {
			wsURL = binanceUFuturesAuthWebsocketURL + "?streams=" + listenKey
			err = b.Websocket.SetWebsocketURL(wsURL, false, false)
			if err != nil {
				return err
			}
		}
	}

	err = b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s",
			b.Name,
			err)
	}

	// if b.Websocket.CanUseAuthenticatedEndpoints() {
	// 	go b.KeepAuthKeyAlive()
	// }

	b.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PongMessage,
		Delay:             pingDelay,
	})

	b.Websocket.Wg.Add(1)
	go b.wsUFuturesReadData()

	// b.setupOrderbookManager()
	subscriptions, err := b.GenerateUFuturesDefaultSubscriptions()
	if err != nil {
		return err
	}

	value, _ := json.Marshal(subscriptions)
	println(string(value))

	return b.SubscribeUFutures(subscriptions)
	// return nil
}

// wsUFuturesReadData receives and passes on websocket messages for processing
// for USDT margined instruments.
func (b *Binance) wsUFuturesReadData() {
	defer b.Websocket.Wg.Done()

	for {
		resp := b.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := b.wsUFuturesHandleData(resp.Raw)
		if err != nil {
			b.Websocket.DataHandler <- err
		}
	}
}

func (b *Binance) wsUFuturesHandleData(respRaw []byte) error {
	println(string(respRaw))
	var multiStreamData map[string]interface{}
	err := json.Unmarshal(respRaw, &multiStreamData)
	if err != nil {
		return err
	}

	if r, ok := multiStreamData["result"]; ok {
		if r == nil {
			return nil
		}
	}

	method, ok := multiStreamData["method"].(string)
	if ok {
		if strings.EqualFold(method, "subscribe") {
			return nil
		}
		if strings.EqualFold(method, "unsubscribe") {
			return nil
		}
	}
	splitted := strings.Split(method, "@")
	if len(splitted) > 1 && splitted[len(splitted)-1] == "arr" {
		//
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

// SubscribeUFutures subscribes to a set of channels
func (b *Binance) SubscribeUFutures(channelsToSubscribe []stream.ChannelSubscription) error {
	return b.handleSubscriptions("SUBSCRIBE", channelsToSubscribe)
}

// UnsubscribeUFutures unsubscribes from a set of channels
func (b *Binance) UnsubscribeUFutures(channelsToUnsubscribe []stream.ChannelSubscription) error {
	return b.handleSubscriptions("UNSUBSCRIBE", channelsToUnsubscribe)
}

func (b *Binance) handleSubscriptions(operation string, subscriptionChannels []stream.ChannelSubscription) error {
	payload := WsPayload{
		ID:     b.Websocket.Conn.GenerateMessageID(false),
		Method: operation,
	}
	for i := range subscriptionChannels {
		payload.Params = append(payload.Params, subscriptionChannels[i].Channel)
		if i%50 == 0 && i != 0 {
			val, _ := json.Marshal(payload)
			println(string(val))
			err := b.Websocket.Conn.SendJSONMessage(payload)
			if err != nil {
				return err
			}
			payload.Params = []string{}
			payload.ID = b.Websocket.Conn.GenerateMessageID(false)
		}
	}
	if len(payload.Params) > 0 {
		val, _ := json.Marshal(payload)
		println(string(val))
		err := b.Websocket.Conn.SendJSONMessage(payload)
		if err != nil {
			return err
		}
	}
	switch operation {
	case "UNSUBSCRIBE":
		b.Websocket.RemoveSubscriptions(subscriptionChannels...)
	case "SUBSCRIBE":
		b.Websocket.AddSuccessfulSubscriptions(subscriptionChannels...)
	}
	return nil
}

// GenerateUFuturesDefaultSubscriptions generates the default subscription set
func (b *Binance) GenerateUFuturesDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = defaultSubscriptions
	var subscriptions []stream.ChannelSubscription
	pairs, err := b.GetEnabledPairs(asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	for z := range channels {
		var subscription stream.ChannelSubscription
		switch channels[z] {
		case assetIndexAllChan, contractInfoAllChan, forceOrderAllChan,
			bookTickerAllChan, tickerAllChan, miniTickerAllChan, markPriceAllChan:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: channels[z],
			})
		case aggTradeChan, depthChan, markPriceChan, tickerChan, klineChan,
			miniTickerChan, bookTickersChan, forceOrderChan, compositeIndexChan, assetIndexChan:
			for y := range pairs {
				lp := pairs[y].Lower()
				lp.Delimiter = ""
				subscription = stream.ChannelSubscription{
					Channel: lp.String() + channels[z],
				}
				switch channels[z] {
				case depthChan:
					subscription.Channel = subscription.Channel + "@100ms"
				case klineChan:
					subscription.Channel = subscription.Channel + "_" + getKlineIntervalString(kline.FiveMin)
				}
				subscriptions = append(subscriptions, subscription)
			}
		default:
			return nil, errors.New("unsupported subscription")
		}
		// switch channels[z] {
		// case depthChan:
		// 	subscription.Channel += "@100ms"
		// case klineChan:
		// 	subscription.Channel += "_" + getKlineIntervalString(kline.FiveMin)
		// case markPriceAllChan:
		// 	subscription.Channel += "@1s"
		// }
		// subscriptions = append(subscriptions, subscription)
	}
	return subscriptions, nil
}
