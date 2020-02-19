package coinbene

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
	"github.com/thrasher-corp/gocryptotrader/log"
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
		err = c.Login()
		if err != nil {
			c.Websocket.DataHandler <- err
			c.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	c.GenerateDefaultSubscriptions()

	return nil
}

// GenerateDefaultSubscriptions generates stuff
func (c *Coinbene) GenerateDefaultSubscriptions() {
	var channels = []string{"orderBook.%s.100", "tradeList.%s", "ticker.%s", "kline.%s"}
	var subscriptions []wshandler.WebsocketChannelSubscription
	pairs, err := c.GetEnabledPairs(asset.PerpetualSwap)
	if err != nil {
		log.Errorf(log.WebsocketMgr, "%s could not generate default subscriptions Err: %s",
			c.Name,
			err)
		return
	}
	for x := range channels {
		for y := range pairs {
			pairs[y].Delimiter = ""
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  fmt.Sprintf(channels[x], pairs[y]),
				Currency: pairs[y],
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
			if string(stream.Raw) == wshandler.Ping {
				err = c.WebsocketConn.SendRawMessage(websocket.TextMessage, []byte(wshandler.Pong))
				if err != nil {
					c.Websocket.DataHandler <- err
				}
				continue
			}
			var result map[string]interface{}
			err = json.Unmarshal(stream.Raw, &result)
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
					c.Websocket.SetCanUseAuthenticatedEndpoints(true)
					c.GenerateAuthSubs()
					continue
				}
				c.Websocket.SetCanUseAuthenticatedEndpoints(false)
				c.Websocket.DataHandler <- fmt.Errorf("message: %s. code: %v", result["message"], result["code"])
				continue
			}

			enabledSwap, err := c.GetEnabledPairs(asset.PerpetualSwap)
			if err != nil {
				c.Websocket.DataHandler <- err
				continue
			}

			format, err := c.GetPairFormat(asset.PerpetualSwap, true)
			if err != nil {
				c.Websocket.DataHandler <- err
				continue
			}

			switch {
			case strings.Contains(result[topic].(string), "ticker"):
				var wsTicker WsTicker
				err = json.Unmarshal(stream.Raw, &wsTicker)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				for x := range wsTicker.Data {
					p, err := currency.NewPairFromFormattedPairs(wsTicker.Data[x].Symbol,
						enabledSwap,
						format)
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					c.Websocket.DataHandler <- &ticker.Price{
						Volume:       wsTicker.Data[x].Volume24h,
						Last:         wsTicker.Data[x].LastPrice,
						High:         wsTicker.Data[x].High24h,
						Low:          wsTicker.Data[x].Low24h,
						Bid:          wsTicker.Data[x].BestBidPrice,
						Ask:          wsTicker.Data[x].BestAskPrice,
						Pair:         p,
						ExchangeName: c.Name,
						AssetType:    asset.PerpetualSwap,
						LastUpdated:  wsTicker.Data[x].Timestamp}
				}
			case strings.Contains(result[topic].(string), "tradeList"):
				var tradeList WsTradeList
				err = json.Unmarshal(stream.Raw, &tradeList)
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
				p := strings.Replace(tradeList.Topic, "tradeList.", "", 1)
				newPair, err := currency.NewPairFromFormattedPairs(p,
					enabledSwap,
					format)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- wshandler.TradeData{
					CurrencyPair: newPair,
					Timestamp:    t,
					Price:        price,
					Amount:       amount,
					Exchange:     c.Name,
					AssetType:    asset.PerpetualSwap,
					Side:         tradeList.Data[0][1],
				}
			case strings.Contains(result[topic].(string), "orderBook"):
				orderBook := struct {
					Topic  string `json:"topic"`
					Action string `json:"action"`
					Data   []struct {
						Bids      [][]string `json:"bids"`
						Asks      [][]string `json:"asks"`
						Version   int64      `json:"version"`
						Timestamp time.Time  `json:"timestamp"`
					} `json:"data"`
				}{}
				err = json.Unmarshal(stream.Raw, &orderBook)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				p := strings.Replace(orderBook.Topic, "orderBook.", "", 1)
				cp, err := currency.NewPairFromFormattedPairs(p,
					enabledSwap,
					format)
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
					newOB.AssetType = asset.PerpetualSwap
					newOB.Pair = cp
					newOB.ExchangeName = c.Name
					newOB.LastUpdated = orderBook.Data[0].Timestamp
					err = c.Websocket.Orderbook.LoadSnapshot(&newOB)
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					c.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: newOB.Pair,
						Asset:    asset.PerpetualSwap,
						Exchange: c.Name,
					}
				} else if orderBook.Action == "update" {
					newOB := wsorderbook.WebsocketOrderbookUpdate{
						Asks:       asks,
						Bids:       bids,
						Asset:      asset.PerpetualSwap,
						Pair:       cp,
						UpdateID:   orderBook.Data[0].Version,
						UpdateTime: orderBook.Data[0].Timestamp,
					}
					err = c.Websocket.Orderbook.Update(&newOB)
					if err != nil {
						c.Websocket.DataHandler <- err
						continue
					}
					c.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: newOB.Pair,
						Asset:    asset.PerpetualSwap,
						Exchange: c.Name,
					}
				}
			case strings.Contains(result[topic].(string), "kline"):
				var kline WsKline
				var tempFloat float64
				var tempKline []float64
				err = json.Unmarshal(stream.Raw, &kline)
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
				p, err := currency.NewPairFromFormattedPairs(kline.Data[0][0].(string),
					enabledSwap,
					format)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- wshandler.KlineData{
					Timestamp:  time.Unix(int64(kline.Data[0][1].(float64)), 0),
					Pair:       p,
					AssetType:  asset.PerpetualSwap,
					Exchange:   c.Name,
					OpenPrice:  tempKline[0],
					ClosePrice: tempKline[1],
					HighPrice:  tempKline[2],
					LowPrice:   tempKline[3],
					Volume:     tempKline[4],
				}
			case strings.Contains(result[topic].(string), "user.account"):
				var userinfo WsUserInfo
				err = json.Unmarshal(stream.Raw, &userinfo)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- userinfo
			case strings.Contains(result[topic].(string), "user.position"):
				var position WsPosition
				err = json.Unmarshal(stream.Raw, &position)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- position
			case strings.Contains(result[topic].(string), "user.order"):
				var orders WsUserOrders
				err = json.Unmarshal(stream.Raw, &orders)
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
	return c.WebsocketConn.SendJSONMessage(sub)
}

// Unsubscribe sends a websocket message to receive data from the channel
func (c *Coinbene) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var sub WsSub
	sub.Operation = "unsubscribe"
	sub.Arguments = []string{channelToSubscribe.Channel}
	return c.WebsocketConn.SendJSONMessage(sub)
}

// Login logs in
func (c *Coinbene) Login() error {
	var sub WsSub
	expTime := time.Now().Add(time.Minute * 10).Format("2006-01-02T15:04:05Z")
	signMsg := expTime + http.MethodGet + "/login"
	tempSign := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(signMsg),
		[]byte(c.API.Credentials.Secret))
	sign := crypto.HexEncodeToString(tempSign)
	sub.Operation = "login"
	sub.Arguments = []string{c.API.Credentials.Key, expTime, sign}
	return c.WebsocketConn.SendJSONMessage(sub)
}
