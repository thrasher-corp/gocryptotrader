package gateio

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	gateioWebsocketEndpoint  = "wss://ws.gateio.ws/v3/"
	gatioWsMethodPing        = "ping"
	gateioWebsocketRateLimit = 120
)

// WsConnect initiates a websocket connection
func (g *Gateio) WsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := g.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	go g.WsHandleData()
	_, err = g.wsServerSignIn()
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v - authentication failed: %v\n", g.Name, err)
		g.Websocket.SetCanUseAuthenticatedEndpoints(false)
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
	signature := crypto.Base64Encode(sigTemp)
	signinWsRequest := WebsocketRequest{
		ID:     g.WebsocketConn.GenerateMessageID(true),
		Method: "server.sign",
		Params: []interface{}{g.API.Credentials.Key, signature, nonce},
	}
	resp, err := g.WebsocketConn.SendMessageReturnResponse(signinWsRequest.ID, signinWsRequest)
	if err != nil {
		g.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return nil, err
	}
	var response WebsocketAuthenticationResponse
	err = json.Unmarshal(resp, &response)
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
				g.Websocket.ReadMessageErrors <- err
				return
			}
			g.Websocket.TrafficAlert <- struct{}{}
			var result WebsocketResponse
			err = json.Unmarshal(resp.Raw, &result)
			if err != nil {
				g.Websocket.DataHandler <- err
				continue
			}

			if result.ID > 0 {
				g.WebsocketConn.AddResponseWithID(result.ID, resp.Raw)
				continue
			}

			if result.Error.Code != 0 {
				if strings.Contains(result.Error.Message, "authentication") {
					g.Websocket.DataHandler <- fmt.Errorf("%v - authentication failed: %v", g.Name, err)
					g.Websocket.SetCanUseAuthenticatedEndpoints(false)
					continue
				}
				g.Websocket.DataHandler <- fmt.Errorf("%v error %s",
					g.Name, result.Error.Message)
				continue
			}

			switch {
			case strings.Contains(result.Method, "ticker"):
				var wsTicker WebsocketTicker
				var c string
				err = json.Unmarshal(result.Params[1], &wsTicker)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = json.Unmarshal(result.Params[0], &c)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				g.Websocket.DataHandler <- &ticker.Price{
					ExchangeName: g.Name,
					Open:         wsTicker.Open,
					Close:        wsTicker.Close,
					Volume:       wsTicker.BaseVolume,
					QuoteVolume:  wsTicker.QuoteVolume,
					High:         wsTicker.High,
					Low:          wsTicker.Low,
					Last:         wsTicker.Last,
					AssetType:    asset.Spot,
					Pair:         currency.NewPairFromString(c),
				}

			case strings.Contains(result.Method, "trades"):
				var trades []WebsocketTrade
				var c string
				err = json.Unmarshal(result.Params[1], &trades)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = json.Unmarshal(result.Params[0], &c)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				for i := range trades {
					g.Websocket.DataHandler <- wshandler.TradeData{
						Timestamp:    time.Now(),
						CurrencyPair: currency.NewPairFromString(c),
						AssetType:    asset.Spot,
						Exchange:     g.Name,
						Price:        trades[i].Price,
						Amount:       trades[i].Amount,
						Side:         trades[i].Type,
					}
				}

			case strings.Contains(result.Method, "depth"):
				var IsSnapshot bool
				var c string
				var data = make(map[string][][]string)
				err = json.Unmarshal(result.Params[0], &IsSnapshot)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = json.Unmarshal(result.Params[2], &c)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				err = json.Unmarshal(result.Params[1], &data)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				var asks, bids []orderbook.Item

				askData, askOk := data["asks"]
				for i := range askData {
					amount, _ := strconv.ParseFloat(askData[i][1], 64)
					price, _ := strconv.ParseFloat(askData[i][0], 64)
					asks = append(asks, orderbook.Item{
						Amount: amount,
						Price:  price,
					})
				}

				bidData, bidOk := data["bids"]
				for i := range bidData {
					amount, _ := strconv.ParseFloat(bidData[i][1], 64)
					price, _ := strconv.ParseFloat(bidData[i][0], 64)
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
					newOrderBook.AssetType = asset.Spot
					newOrderBook.Pair = currency.NewPairFromString(c)
					newOrderBook.ExchangeName = g.Name

					err = g.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
					if err != nil {
						g.Websocket.DataHandler <- err
					}
				} else {
					err = g.Websocket.Orderbook.Update(
						&wsorderbook.WebsocketOrderbookUpdate{
							Asks:       asks,
							Bids:       bids,
							Pair:       currency.NewPairFromString(c),
							UpdateTime: time.Now(),
							Asset:      asset.Spot,
						})
					if err != nil {
						g.Websocket.DataHandler <- err
					}
				}

				g.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
					Pair:     currency.NewPairFromString(c),
					Asset:    asset.Spot,
					Exchange: g.Name,
				}

			case strings.Contains(result.Method, "kline"):
				var data []interface{}
				err = json.Unmarshal(result.Params[0], &data)
				if err != nil {
					g.Websocket.DataHandler <- err
					continue
				}

				open, _ := strconv.ParseFloat(data[1].(string), 64)
				closePrice, _ := strconv.ParseFloat(data[2].(string), 64)
				high, _ := strconv.ParseFloat(data[3].(string), 64)
				low, _ := strconv.ParseFloat(data[4].(string), 64)
				volume, _ := strconv.ParseFloat(data[5].(string), 64)

				g.Websocket.DataHandler <- wshandler.KlineData{
					Timestamp:  time.Now(),
					Pair:       currency.NewPairFromString(data[7].(string)),
					AssetType:  asset.Spot,
					Exchange:   g.Name,
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
	var subscriptions []wshandler.WebsocketChannelSubscription
	enabledCurrencies := g.GetEnabledPairs(asset.Spot)
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
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
	var subscriptions []wshandler.WebsocketChannelSubscription
	enabledCurrencies := g.GetEnabledPairs(asset.Spot)
	for i := range channels {
		for j := range enabledCurrencies {
			params := make(map[string]interface{})
			if strings.EqualFold(channels[i], "depth.subscribe") {
				params["limit"] = 30
				params["interval"] = "0.1"
			} else if strings.EqualFold(channels[i], "kline.subscribe") {
				params["interval"] = 1800
			}
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
				Params:   params,
			})
		}
	}
	g.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (g *Gateio) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	params := []interface{}{g.FormatExchangeCurrency(channelToSubscribe.Currency,
		asset.Spot).Upper()}

	for i := range channelToSubscribe.Params {
		params = append(params, channelToSubscribe.Params[i])
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
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return err
	}
	if response.Result.Status != "success" {
		return fmt.Errorf("%v could not subscribe to %v", g.Name, channelToSubscribe.Channel)
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (g *Gateio) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	unsbuscribeText := strings.Replace(channelToSubscribe.Channel, "subscribe", "unsubscribe", 1)
	subscribe := WebsocketRequest{
		ID:     g.WebsocketConn.GenerateMessageID(true),
		Method: unsbuscribeText,
		Params: []interface{}{g.FormatExchangeCurrency(channelToSubscribe.Currency,
			asset.Spot).Upper(), 1800},
	}
	resp, err := g.WebsocketConn.SendMessageReturnResponse(subscribe.ID, subscribe)
	if err != nil {
		return err
	}
	var response WebsocketAuthenticationResponse
	err = json.Unmarshal(resp, &response)
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
	err = json.Unmarshal(resp, &balance)
	if err != nil {
		return &balance, err
	}

	return &balance, nil
}

func (g *Gateio) wsGetOrderInfo(market string, offset, limit int) (*WebSocketOrderQueryResult, error) {
	if !g.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, fmt.Errorf("%v not authorised to get order info", g.Name)
	}
	ord := WebsocketRequest{
		ID:     g.WebsocketConn.GenerateMessageID(true),
		Method: "order.query",
		Params: []interface{}{
			market,
			offset,
			limit,
		},
	}
	resp, err := g.WebsocketConn.SendMessageReturnResponse(ord.ID, ord)
	if err != nil {
		return nil, err
	}
	var orderQuery WebSocketOrderQueryResult
	err = json.Unmarshal(resp, &orderQuery)
	if err != nil {
		return &orderQuery, err
	}
	return &orderQuery, nil
}
