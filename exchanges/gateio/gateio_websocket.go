package gateio

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ws/connection"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ws/monitor"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	gateioWebsocketEndpoint  = "wss://ws.gate.io/v3/"
	gatioWsMethodPing        = "ping"
	gateioWebsocketRateLimit = 120
)

// WsConnect initiates a websocket connection
func (g *Gateio) WsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(monitor.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := g.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	go g.WsHandleData()
	_, err = g.wsServerSignIn()
	if err != nil {
		log.Errorf("%v - authentication failed: %v", g.Name, err)
	}
	g.GenerateAuthenticatedSubscriptions()
	g.GenerateDefaultSubscriptions()
	return nil
}

func (g *Gateio) wsServerSignIn() (*WebsocketAuthenticationResponse, error) {
	if !g.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		return nil, fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", g.Name)
	}
	nonce := int(time.Now().Unix() * 1000)
	sigTemp := g.GenerateSignature(strconv.Itoa(nonce))
	signature := common.Base64Encode(sigTemp)
	signinWsRequest := WebsocketRequest{
		ID:     g.WebsocketConn.GenerateMessageID(true),
		Method: "server.sign",
		Params: []interface{}{g.APIKey, signature, nonce},
	}
	resp, err := g.WebsocketConn.SendMessageReturnResponse(signinWsRequest.ID, signinWsRequest)
	if err != nil {
		g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return nil, err
	}
	var response WebsocketAuthenticationResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return nil, err
	}
	if response.Result.Status == "success" {
		g.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	return &response, nil
}

// WsHandleData handles all the websocket data coming from the websocket
// connection
func (g *Gateio) WsHandleData() {
	g.Websocket.Wg.Add(1)

	defer func() {
		g.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-g.Websocket.ShutdownC:
			return

		default:
			resp, err := g.WebsocketConn.ReadMessage()
			if err != nil {
				g.Websocket.DataHandler <- err
				return
			}
			g.Websocket.TrafficAlert <- struct{}{}
			var result WebsocketResponse
			err = common.JSONDecode(resp.Raw, &result)
			if err != nil {
				g.Websocket.DataHandler <- err
				continue
			}

			if result.ID > 0 {
				g.WebsocketConn.AddResponseWithID(result.ID, resp.Raw)
				continue
			}

			if result.Error.Code != 0 {
				if common.StringContains(result.Error.Message, "authentication") {
					g.Websocket.DataHandler <- fmt.Errorf("%v - authentication failed: %v", g.Name, err)
					g.Websocket.SetCanUseAuthenticatedEndpoints(false)
					continue
				}
				g.Websocket.DataHandler <- fmt.Errorf("%v error %s",
					g.Name, result.Error.Message)
				continue
			}

			switch {
			case common.StringContains(result.Method, "ticker"):
				var ticker WebsocketTicker
				var c string
				err = common.JSONDecode(result.Params[1], &ticker)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = common.JSONDecode(result.Params[0], &c)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				g.Websocket.DataHandler <- monitor.TickerData{
					Timestamp:  time.Now(),
					Pair:       currency.NewPairFromString(c),
					AssetType:  "SPOT",
					Exchange:   g.GetName(),
					ClosePrice: ticker.Close,
					Quantity:   ticker.BaseVolume,
					OpenPrice:  ticker.Open,
					HighPrice:  ticker.High,
					LowPrice:   ticker.Low,
				}

			case common.StringContains(result.Method, "trades"):
				var trades []WebsocketTrade
				var c string
				err = common.JSONDecode(result.Params[1], &trades)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = common.JSONDecode(result.Params[0], &c)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				for _, trade := range trades {
					g.Websocket.DataHandler <- monitor.TradeData{
						Timestamp:    time.Now(),
						CurrencyPair: currency.NewPairFromString(c),
						AssetType:    "SPOT",
						Exchange:     g.GetName(),
						Price:        trade.Price,
						Amount:       trade.Amount,
						Side:         trade.Type,
					}
				}

			case common.StringContains(result.Method, "depth"):
				var IsSnapshot bool
				var c string
				var data = make(map[string][][]string)
				err = common.JSONDecode(result.Params[0], &IsSnapshot)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = common.JSONDecode(result.Params[2], &c)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = common.JSONDecode(result.Params[1], &data)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				var asks, bids []orderbook.Item

				askData, askOk := data["asks"]
				for _, ask := range askData {
					amount, _ := strconv.ParseFloat(ask[1], 64)
					price, _ := strconv.ParseFloat(ask[0], 64)
					asks = append(asks, orderbook.Item{
						Amount: amount,
						Price:  price,
					})
				}

				bidData, bidOk := data["bids"]
				for _, bid := range bidData {
					amount, _ := strconv.ParseFloat(bid[1], 64)
					price, _ := strconv.ParseFloat(bid[0], 64)
					bids = append(bids, orderbook.Item{
						Amount: amount,
						Price:  price,
					})
				}

				if !askOk && !bidOk {
					g.Websocket.DataHandler <- errors.New("gatio websocket error - cannot access ask or bid data")
				}

				if IsSnapshot {
					if !askOk {
						g.Websocket.DataHandler <- errors.New("gatio websocket error - cannot access ask data")
					}

					if !bidOk {
						g.Websocket.DataHandler <- errors.New("gatio websocket error - cannot access bid data")
					}

					var newOrderBook orderbook.Base
					newOrderBook.Asks = asks
					newOrderBook.Bids = bids
					newOrderBook.AssetType = "SPOT"
					newOrderBook.Pair = currency.NewPairFromString(c)

					err = g.Websocket.Orderbook.LoadSnapshot(&newOrderBook,
						g.GetName(),
						false)
					if err != nil {
						g.Websocket.DataHandler <- err
					}
				} else {
					err = g.Websocket.Orderbook.Update(asks,
						bids,
						currency.NewPairFromString(c),
						time.Now(),
						g.GetName(),
						"SPOT")
					if err != nil {
						g.Websocket.DataHandler <- err
					}
				}

				g.Websocket.DataHandler <- monitor.WebsocketOrderbookUpdate{
					Pair:     currency.NewPairFromString(c),
					Asset:    "SPOT",
					Exchange: g.GetName(),
				}

			case common.StringContains(result.Method, "kline"):
				var data []interface{}
				err = common.JSONDecode(result.Params[0], &data)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				open, _ := strconv.ParseFloat(data[1].(string), 64)
				closePrice, _ := strconv.ParseFloat(data[2].(string), 64)
				high, _ := strconv.ParseFloat(data[3].(string), 64)
				low, _ := strconv.ParseFloat(data[4].(string), 64)
				volume, _ := strconv.ParseFloat(data[5].(string), 64)

				g.Websocket.DataHandler <- monitor.KlineData{
					Timestamp:  time.Now(),
					Pair:       currency.NewPairFromString(data[7].(string)),
					AssetType:  "SPOT",
					Exchange:   g.GetName(),
					OpenPrice:  open,
					ClosePrice: closePrice,
					HighPrice:  high,
					LowPrice:   low,
					Volume:     volume,
				}
			}
		}
	}
}

// GenerateAuthenticatedSubscriptions Adds authenticated subscriptions to websocket to be handled by ManageSubscriptions()
func (g *Gateio) GenerateAuthenticatedSubscriptions() {
	if !g.Websocket.CanUseAuthenticatedEndpoints() {
		return
	}
	var channels = []string{"balance.subscribe", "order.subscribe"}
	var subscriptions []monitor.WebsocketChannelSubscription
	enabledCurrencies := g.GetEnabledCurrencies()
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, monitor.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	g.Websocket.SubscribeToChannels(subscriptions)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (g *Gateio) GenerateDefaultSubscriptions() {
	var channels = []string{"ticker.subscribe", "trades.subscribe", "depth.subscribe", "kline.subscribe"}
	var subscriptions []monitor.WebsocketChannelSubscription
	enabledCurrencies := g.GetEnabledCurrencies()
	for i := range channels {
		for j := range enabledCurrencies {
			params := make(map[string]interface{})
			if strings.EqualFold(channels[i], "depth.subscribe") {
				params["limit"] = 30
				params["interval"] = "0.1"
			} else if strings.EqualFold(channels[i], "kline.subscribe") {
				params["interval"] = 1800
			}
			subscriptions = append(subscriptions, monitor.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
				Params:   params,
			})
		}
	}
	g.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (g *Gateio) Subscribe(channelToSubscribe monitor.WebsocketChannelSubscription) error {
	params := []interface{}{channelToSubscribe.Currency.String()}
	for _, paramValue := range channelToSubscribe.Params {
		params = append(params, paramValue)
	}

	subscribe := WebsocketRequest{
		ID:     g.WebsocketConn.GenerateMessageID(true),
		Method: channelToSubscribe.Channel,
		Params: params,
	}

	resp, err := g.WebsocketConn.SendMessageReturnResponse(subscribe.ID, subscribe)
	if err != nil {
		return err
	}
	var response WebsocketAuthenticationResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return err
	}
	if response.Result.Status != "success" {
		return fmt.Errorf("%v could not subscribe to %v", g.Name, channelToSubscribe.Channel)
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) Unsubscribe(channelToSubscribe monitor.WebsocketChannelSubscription) error {
	unsbuscribeText := strings.Replace(channelToSubscribe.Channel, "subscribe", "unsubscribe", 1)
	subscribe := WebsocketRequest{
		ID:     g.WebsocketConn.GenerateMessageID(true),
		Method: unsbuscribeText,
		Params: []interface{}{channelToSubscribe.Currency.String(), 1800},
	}
	resp, err := g.WebsocketConn.SendMessageReturnResponse(subscribe.ID, subscribe)
	if err != nil {
		return err
	}
	var response WebsocketAuthenticationResponse
	err = common.JSONDecode(resp, &response)
	if err != nil {
		return err
	}
	if response.Result.Status != "success" {
		return fmt.Errorf("%v could not subscribe to %v", g.Name, channelToSubscribe.Channel)
	}
	return nil
}

func (g *Gateio) wsGetBalance(currencies []string) (*WsGetBalanceResponse, error) {
	if !g.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to get balance", g.Name)
	}
	balanceWsRequest := wsGetBalanceRequest{
		ID:     g.WebsocketConn.GenerateMessageID(false),
		Method: "balance.query",
		Params: currencies,
	}
	resp, err := g.WebsocketConn.SendMessageReturnResponse(balanceWsRequest.ID, balanceWsRequest)
	if err != nil {
		return nil, err
	}
	var balance WsGetBalanceResponse
	err = common.JSONDecode(resp, &balance)
	if err != nil {
		return &balance, err
	}

	return &balance, nil
}

func (g *Gateio) wsGetOrderInfo(market string, offset, limit int) (*WebSocketOrderQueryResult, error) {
	if !g.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to get order info", g.Name)
	}
	order := WebsocketRequest{
		ID:     g.WebsocketConn.GenerateMessageID(true),
		Method: "order.query",
		Params: []interface{}{
			market,
			offset,
			limit,
		},
	}
	resp, err := g.WebsocketConn.SendMessageReturnResponse(order.ID, order)
	if err != nil {
		return nil, err
	}
	var orderQuery WebSocketOrderQueryResult
	err = common.JSONDecode(resp, &orderQuery)
	if err != nil {
		return &orderQuery, err
	}
	return &orderQuery, nil
}
