package coinbene

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
)

const (
	coinbeneWsURL = "wss://ws-contract.coinbene.vip/openapi/ws"
	event         = "event"
	topic         = "topic"
)

// WsConnect connects to websocket
func (c *Coinbene) WsConnect() error {
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := c.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	go c.WsDataHandler()
	if c.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		c.Login()
	}
	c.GenerateDefaultSubscriptions()

	return nil
}

// GenerateDefaultSubscriptions generates stuff
func (c *Coinbene) GenerateDefaultSubscriptions() {
	var channels = []string{"orderBook.%s.100", "tradeList.%s", "ticker.%s", "kline.%s"}
	var subscriptions []wshandler.WebsocketChannelSubscription
	for x := range channels {
		for y := range c.EnabledPairs {
			c.EnabledPairs[y].Delimiter = ""
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  fmt.Sprintf(channels[x], c.EnabledPairs[y]),
				Currency: c.EnabledPairs[y],
			})
		}
	}
	c.Websocket.SubscribeToChannels(subscriptions)
}

// GenerateAuthSubs generates auth subs
func (c *Coinbene) GenerateAuthSubs() {
	var subscriptions []wshandler.WebsocketChannelSubscription
	var sub wshandler.WebsocketChannelSubscription
	var userChannels = []string{"user.account", "user.position", "user.order"}
	for z := range userChannels {
		sub.Channel = userChannels[z]
		subscriptions = append(subscriptions, sub)
	}
	c.Websocket.SubscribeToChannels(subscriptions)
}

// WsDataHandler handles websocket data
func (c *Coinbene) WsDataHandler() {
	c.Websocket.Wg.Add(1)

	defer c.Websocket.Wg.Done()

	for {
		select {
		case <-c.Websocket.ShutdownC:
			return

		default:
			stream, err := c.WebsocketConn.ReadMessage()
			if err != nil {
				c.Websocket.DataHandler <- err
				return
			}
			c.Websocket.TrafficAlert <- struct{}{}
			if string(stream.Raw) == "ping" {
				c.WebsocketConn.Lock()
				c.WebsocketConn.Connection.WriteMessage(websocket.TextMessage, []byte("pong"))
				c.WebsocketConn.Unlock()
				continue
			}
			var result map[string]interface{}
			err = common.JSONDecode(stream.Raw, &result)
			if err != nil {
				c.Websocket.DataHandler <- err
			}
			_, ok := result[event]
			switch {
			case ok && (result[event].(string) == "subscribe" || result[event].(string) == "unsubscribe"):
				continue
			case ok && result[event].(string) == "error":
				c.Websocket.DataHandler <- fmt.Errorf("message: %s. code: %v", result["message"], result["code"])
				continue
			}
			if ok && strings.Contains(result[event].(string), "login") {
				if result["success"].(bool) {
					c.GenerateAuthSubs()
				}
				c.AuthenticatedWebsocketAPISupport = false
				c.Websocket.DataHandler <- fmt.Errorf("message: %s. code: %v", result["message"], result["code"])
				continue
			}
			switch {
			case strings.Contains(result[topic].(string), "ticker"):
				var ticker WsTicker
				err = common.JSONDecode(stream.Raw, &ticker)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				for x := range ticker.Data {
					c.Websocket.DataHandler <- wshandler.TickerData{
						Quantity:   ticker.Data[x].Volume24h,
						ClosePrice: ticker.Data[x].LastPrice,
						HighPrice:  ticker.Data[x].High24h,
						LowPrice:   ticker.Data[x].Low24h,
						Pair:       currency.NewPairFromString(ticker.Data[x].Symbol),
						Exchange:   c.Name,
						AssetType:  orderbook.Swap,
					}
				}
			case strings.Contains(result[topic].(string), "tradeList"):
				var tradeList WsTradeList
				err = common.JSONDecode(stream.Raw, &tradeList)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				var t time.Time
				var price, amount float64
				t, err = time.Parse(time.RFC3339, tradeList.Data[0][3])
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				price, err = strconv.ParseFloat(tradeList.Data[0][0], 64)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				amount, err = strconv.ParseFloat(tradeList.Data[0][2], 64)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- wshandler.TradeData{
					CurrencyPair: currency.NewPairFromString(strings.Replace(tradeList.Topic, "tradeList.", "", 1)),
					Timestamp:    t,
					Price:        price,
					Amount:       amount,
					Exchange:     c.Name,
					AssetType:    orderbook.Swap,
					Side:         tradeList.Data[0][1],
				}
			case strings.Contains(result[topic].(string), "orderBook"):
				var orderBook WsOrderbook
				err = common.JSONDecode(stream.Raw, &orderBook)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				var amount, price float64
				var asks, bids []orderbook.Item
				for i := range orderBook.Data[0].Asks {
					amount, err = strconv.ParseFloat(orderBook.Data[0].Asks[i][1], 64)
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					price, err = strconv.ParseFloat(orderBook.Data[0].Asks[i][0], 64)
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					asks = append(asks, orderbook.Item{
						Amount: amount,
						Price:  price,
					})
				}
				for j := range orderBook.Data[0].Bids {
					amount, err = strconv.ParseFloat(orderBook.Data[0].Bids[j][1], 64)
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					price, err = strconv.ParseFloat(orderBook.Data[0].Bids[j][0], 64)
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					bids = append(bids, orderbook.Item{
						Amount: amount,
						Price:  price,
					})
				}
				if orderBook.Action == "insert" {
					var newOB orderbook.Base
					newOB.Asks = asks
					newOB.Bids = bids
					newOB.AssetType = orderbook.Swap
					newOB.Pair = currency.NewPairFromString(strings.Replace(orderBook.Topic, "tradeList.", "", 1))
					newOB.ExchangeName = c.Name
					err = c.Websocket.Orderbook.LoadSnapshot(&newOB, true)
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					c.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: newOB.Pair,
						Asset:    orderbook.Swap,
						Exchange: c.Name,
					}
				} else if orderBook.Action == "update" {
					newOB := wsorderbook.WebsocketOrderbookUpdate{
						Asks:         asks,
						Bids:         bids,
						AssetType:    orderbook.Swap,
						CurrencyPair: currency.NewPairFromString(strings.Replace(orderBook.Topic, "tradeList.", "", 1)),
						UpdateID:     orderBook.Version,
					}
					err = c.Websocket.Orderbook.Update(&newOB)
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					c.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: newOB.CurrencyPair,
						Asset:    orderbook.Swap,
						Exchange: c.Name,
					}
				}
			case strings.Contains(result[topic].(string), "kline"):
				var kline WsKline
				var tempFloat float64
				var tempKline []float64
				err = common.JSONDecode(stream.Raw, &kline)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				for x := 2; x < len(kline.Data[0]); x++ {
					tempFloat, err = strconv.ParseFloat(kline.Data[0][x].(string), 64)
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					tempKline = append(tempKline, tempFloat)
				}
				c.Websocket.DataHandler <- wshandler.KlineData{
					Timestamp:  time.Unix(int64(kline.Data[0][1].(float64)), 0),
					Pair:       currency.NewPairFromString(kline.Data[0][0].(string)),
					AssetType:  orderbook.Swap,
					Exchange:   c.Name,
					OpenPrice:  tempKline[0],
					ClosePrice: tempKline[1],
					HighPrice:  tempKline[2],
					LowPrice:   tempKline[3],
					Volume:     tempKline[4],
				}
			case strings.Contains(result[topic].(string), "user.account"):
				var userinfo WsUserInfo
				err = common.JSONDecode(stream.Raw, &userinfo)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- userinfo
			case strings.Contains(result[topic].(string), "user.position"):
				var position WsPosition
				err = common.JSONDecode(stream.Raw, &position)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- position
			case strings.Contains(result[topic].(string), "user.order"):
				var orders WsUserOrders
				err = common.JSONDecode(stream.Raw, &orders)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- orders
			default:
				c.Websocket.DataHandler <- fmt.Errorf("%s - unhandled response '%s'", c.Name, stream.Raw)
			}
		}
	}
}

// Subscribe sends a websocket message to receive data from the channel
func (c *Coinbene) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var sub WsSub
	sub.Operation = "subscribe"
	sub.Arguments = []string{channelToSubscribe.Channel}
	return c.WebsocketConn.SendMessage(sub)
}

// Unsubscribe sends a websocket message to receive data from the channel
func (c *Coinbene) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var sub WsSub
	sub.Operation = "unsubscribe"
	sub.Arguments = []string{channelToSubscribe.Channel}
	return c.WebsocketConn.SendMessage(sub)
}

// Login logs in
func (c *Coinbene) Login() error {
	var sub WsSub
	expTime := time.Now().Add(time.Minute * 10).Format("2006-01-02T15:04:05Z")
	signMsg := expTime + http.MethodGet + "/login"
	tempSign := common.GetHMAC(common.HashSHA256, []byte(signMsg), []byte(c.APISecret))
	sign := common.HexEncodeToString(tempSign)
	sub.Operation = "login"
	sub.Arguments = []string{c.APIKey, expTime, sign}
	return c.WebsocketConn.SendMessage(sub)
}
