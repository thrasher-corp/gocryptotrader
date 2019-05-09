package gateio

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	gateioWebsocketEndpoint = "wss://ws.gate.io/v3/"
	gatioWsMethodPing       = "ping"
)

// WsConnect initiates a websocket connection
func (g *Gateio) WsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	if g.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(g.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	g.WebsocketConn, _, err = dialer.Dial(g.Websocket.GetWebsocketURL(),
		http.Header{})
	if err != nil {
		return err
	}

	if g.AuthenticatedAPISupport {
		err = g.wsServerSignIn()
		if err != nil {
			log.Errorf("%v - wsServerSignin() failed: %v", g.GetName(), err)
		}
		time.Sleep(time.Second * 2) // sleep to allow server to complete sign-on if further authenticated requests are sent piror to this they will fail
	}

	go g.WsHandleData()
	g.GenerateDefaultSubscriptions()


	return nil
}

func (g *Gateio) wsServerSignIn() error {
	nonce := int(time.Now().Unix() * 1000)
	sigTemp := g.GenerateSignature(strconv.Itoa(nonce))
	signature := common.Base64Encode(sigTemp)
	signinWsRequest := WebsocketRequest{
		ID:     IDSignIn,
		Method: "server.sign",
		Params: []interface{}{g.APIKey, signature, nonce},
	}
	return g.WebsocketConn.WriteJSON(signinWsRequest)
}

// WsReadData reads from the websocket connection and returns the websocket
// response
func (g *Gateio) WsReadData() (exchange.WebsocketResponse, error) {
	_, resp, err := g.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	g.Websocket.TrafficAlert <- struct{}{}
	return exchange.WebsocketResponse{Raw: resp}, nil
}

// WsHandleData handles all the websocket data coming from the websocket
// connection
func (g *Gateio) WsHandleData() {
	g.Websocket.Wg.Add(1)

	defer func() {
		err := g.WebsocketConn.Close()
		if err != nil {
			g.Websocket.DataHandler <- fmt.Errorf("gateio_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		g.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-g.Websocket.ShutdownC:
			return

		default:
			resp, err := g.WsReadData()
			if err != nil {
				g.Websocket.DataHandler <- err
				// Read data errpr messages can overwhelm and panic the application
				time.Sleep(time.Second)
				continue
			}

			var result WebsocketResponse
			err = common.JSONDecode(resp.Raw, &result)
			if err != nil {
				g.Websocket.DataHandler <- err
				continue
			}

			if result.Error.Code != 0 {
				if common.StringContains(result.Error.Message, "authentication") {
					g.Websocket.DataHandler <- fmt.Errorf("%v - WebSocket authentication failed ",
						g.GetName())
					g.AuthenticatedAPISupport = false
					continue
				}
				g.Websocket.DataHandler <- fmt.Errorf("gateio_websocket.go error %s",
					result.Error.Message)
				continue
			}

			switch result.ID {
			case IDBalance:
				var balance WebsocketBalance
				var balanceInterface interface{}
				err = json.Unmarshal(result.Result, &balanceInterface)
				if err != nil {
					g.Websocket.DataHandler <- err
				}
				var p WebsocketBalanceCurrency
				switch x := balanceInterface.(type) {
				case map[string]interface{}:
					for xx := range x {
						switch kk := x[xx].(type) {
						case map[string]interface{}:
							p = WebsocketBalanceCurrency{
								Currency:  xx,
								Available: kk["available"].(string),
								Locked:    kk["freeze"].(string),
							}
							balance.Currency = append(balance.Currency, p)
						default:
							break
						}
					}
				default:
					break
				}
				g.Websocket.DataHandler <- balance
			case IDOrderQuery:
				var orderQuery WebSocketOrderQueryResult
				err = common.JSONDecode(result.Result, &orderQuery)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}
				g.Websocket.DataHandler <- orderQuery
			default:
				break
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

				g.Websocket.DataHandler <- exchange.TickerData{
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
					g.Websocket.DataHandler <- exchange.TradeData{
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

				g.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
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

				g.Websocket.DataHandler <- exchange.KlineData{
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

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (g *Gateio) GenerateDefaultSubscriptions() {
	var channels = []string{"ticker.subscribe", "trades.subscribe", "depth.subscribe", "kline.subscribe"}
	enabledCurrencies := g.GetEnabledCurrencies()
	for i := range channels {
		for j := range enabledCurrencies {
			params := make(map[string]interface{})
			if strings.EqualFold(channels[i], "depth.subscribe") {
				params["what"] = 30
				params["who"] = "0.1"
			} else if strings.EqualFold(channels[i], "kline.subscribe") {
				params["sup"] = 1800
			}
			g.Websocket.ChannelsToSubscribe = append(g.Websocket.ChannelsToSubscribe, exchange.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
				Params:   params,
			})
		}
	}
}

// Subscribe tells the websocket connection monitor to not bother with Binance
// Subscriptions are URL argument based and have no need to sub/unsub from channels
func (g *Gateio) Subscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	params := []interface{}{channelToSubscribe.Currency.String()}
	for _, paramValue := range channelToSubscribe.Params {
		params = append(params, paramValue)
	}
	subscribe := WebsocketRequest{
		ID:     1337,
		Method: channelToSubscribe.Channel,
		Params: params,
	}

	data, err := common.JSONEncode(subscribe)
	if err != nil {
		return err
	}

	time.Sleep(30 * time.Millisecond)
	return g.WebsocketConn.WriteMessage(websocket.TextMessage, data)
}

// Unsubscribe tells the websocket connection monitor to not bother with Binance
// Subscriptions are URL argument based and have no need to sub/unsub from channels
func (g *Gateio) Unsubscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	unsbuscribeText := strings.Replace(channelToSubscribe.Channel, "subscribe", "unsubscribe", 1)
	subscribe := WebsocketRequest{
		ID:     1337,
		Method: unsbuscribeText,
		Params: []interface{}{channelToSubscribe.Currency.String(), 1800},
	}

	data, err := common.JSONEncode(subscribe)
	if err != nil {
		return err
	}

	time.Sleep(30 * time.Millisecond)
	return g.WebsocketConn.WriteMessage(websocket.TextMessage, data)
}

func (g *Gateio) wsGetBalance() error {
	balanceWsRequest := WebsocketRequest{
		ID:     IDBalance,
		Method: "balance.query",
		Params: []interface{}{},
	}
	return g.WebsocketConn.WriteJSON(balanceWsRequest)
}

func (g *Gateio) wsGetOrderInfo(market string, offset, limit int) error {
	order := WebsocketRequest{
		ID:     IDOrderQuery,
		Method: "order.query",
		Params: []interface{}{
			market,
			offset,
			limit,
		},
	}
	return g.WebsocketConn.WriteJSON(order)
}
