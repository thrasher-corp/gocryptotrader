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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
)

const (
	wsContractURL = "wss://ws-contract.coinbene.vip/openapi/ws"
	event         = "event"
	topic         = "topic"
)

var comms = make(chan wshandler.WebsocketResponse)

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

	go c.wsReadData()
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
	pairs := c.GetEnabledPairs(asset.PerpetualSwap)
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

// wsReadData receives and passes on websocket messages for processing
func (c *Coinbene) wsReadData() {
	c.Websocket.Wg.Add(1)
	defer c.Websocket.Wg.Done()
	for {
		select {
		case <-c.Websocket.ShutdownC:
			return
		default:
			resp, err := c.WebsocketConn.ReadMessage()
			if err != nil {
				c.Websocket.ReadMessageErrors <- err
				return
			}
			err = c.wsHandleData(resp.Raw)
			if err != nil {
				c.Websocket.DataHandler <- err
			}
		}
	}
}

func (c *Coinbene) wsHandleData(respRaw []byte) error {
	c.Websocket.TrafficAlert <- struct{}{}
	if string(respRaw) == wshandler.Ping {
		err := c.WebsocketConn.SendRawMessage(websocket.TextMessage, []byte(wshandler.Pong))
		if err != nil {
			return err
		}
		return nil
	}
	var result map[string]interface{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	_, ok := result[event]
	switch {
	case ok && (result[event].(string) == "subscribe" || result[event].(string) == "unsubscribe"):
		return nil
	case ok && result[event].(string) == "error":
		return fmt.Errorf("message: %s. code: %v", result["message"], result["code"])
	}
	if ok && strings.Contains(result[event].(string), "login") {
		if result["success"].(bool) {
			c.Websocket.SetCanUseAuthenticatedEndpoints(true)
			c.GenerateAuthSubs()
			return nil
		}
		c.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return fmt.Errorf("message: %s. code: %v", result["message"], result["code"])
	}
	switch {
	case strings.Contains(result[topic].(string), "ticker"):
		var wsTicker WsTicker
		err = json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}
		for x := range wsTicker.Data {
			c.Websocket.DataHandler <- &ticker.Price{
				Volume: wsTicker.Data[x].Volume24h,
				Last:   wsTicker.Data[x].LastPrice,
				High:   wsTicker.Data[x].High24h,
				Low:    wsTicker.Data[x].Low24h,
				Bid:    wsTicker.Data[x].BestBidPrice,
				Ask:    wsTicker.Data[x].BestAskPrice,
				Pair: currency.NewPairFromFormattedPairs(wsTicker.Data[x].Symbol,
					c.GetEnabledPairs(asset.PerpetualSwap),
					c.GetPairFormat(asset.PerpetualSwap, true)),
				ExchangeName: c.Name,
				AssetType:    asset.PerpetualSwap,
				LastUpdated:  wsTicker.Data[x].Timestamp,
			}
		}
	case strings.Contains(result[topic].(string), "tradeList"):
		var tradeList WsTradeList
		err = json.Unmarshal(respRaw, &tradeList)
		if err != nil {
			return err
		}
		var t time.Time
		var price, amount float64
		t, err = time.Parse(time.RFC3339, tradeList.Data[0][3])
		if err != nil {
			return err
		}
		price, err = strconv.ParseFloat(tradeList.Data[0][0], 64)
		if err != nil {
			return err
		}
		amount, err = strconv.ParseFloat(tradeList.Data[0][2], 64)
		if err != nil {
			return err
		}
		p := strings.Replace(tradeList.Topic, "tradeList.", "", 1)
		var tSide = order.Buy
		if tradeList.Data[0][1] == "s" {
			tSide = order.Sell
		}
		c.Websocket.DataHandler <- wshandler.TradeData{
			CurrencyPair: currency.NewPairFromFormattedPairs(p,
				c.GetEnabledPairs(asset.PerpetualSwap),
				c.GetPairFormat(asset.PerpetualSwap, true)),
			Timestamp: t,
			Price:     price,
			Amount:    amount,
			Exchange:  c.Name,
			AssetType: asset.PerpetualSwap,
			Side:      tSide,
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
		err = json.Unmarshal(respRaw, &orderBook)
		if err != nil {
			return err
		}
		p := strings.Replace(orderBook.Topic, "orderBook.", "", 1)
		cp := currency.NewPairFromFormattedPairs(p,
			c.GetEnabledPairs(asset.PerpetualSwap),
			c.GetPairFormat(asset.PerpetualSwap, true))
		var amount, price float64
		var asks, bids []orderbook.Item
		for i := range orderBook.Data[0].Asks {
			amount, err = strconv.ParseFloat(orderBook.Data[0].Asks[i][1], 64)
			if err != nil {
				return err
			}
			price, err = strconv.ParseFloat(orderBook.Data[0].Asks[i][0], 64)
			if err != nil {
				return err
			}
			asks = append(asks, orderbook.Item{
				Amount: amount,
				Price:  price,
			})
		}
		for j := range orderBook.Data[0].Bids {
			amount, err = strconv.ParseFloat(orderBook.Data[0].Bids[j][1], 64)
			if err != nil {
				return err
			}
			price, err = strconv.ParseFloat(orderBook.Data[0].Bids[j][0], 64)
			if err != nil {
				return err
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
				return err
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
				return err
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
		err = json.Unmarshal(respRaw, &kline)
		if err != nil {
			return err
		}
		for x := 2; x < len(kline.Data[0]); x++ {
			tempFloat, err = strconv.ParseFloat(kline.Data[0][x].(string), 64)
			if err != nil {
				return err
			}
			tempKline = append(tempKline, tempFloat)
		}
		p := currency.NewPairFromFormattedPairs(kline.Data[0][0].(string),
			c.GetEnabledPairs(asset.PerpetualSwap),
			c.GetPairFormat(asset.PerpetualSwap, true))
		if tempKline == nil && len(tempKline) < 5 {
			return errors.New(c.Name + " - received bad data ")
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
		var userInfo WsUserInfo
		err = json.Unmarshal(respRaw, &userInfo)
		if err != nil {
			return err
		}
		c.Websocket.DataHandler <- userInfo
	case strings.Contains(result[topic].(string), "user.position"):
		var position WsPosition
		err = json.Unmarshal(respRaw, &position)
		if err != nil {
			return err
		}
		c.Websocket.DataHandler <- position
	case strings.Contains(result[topic].(string), "user.order"):
		var orders WsUserOrders
		err = json.Unmarshal(respRaw, &orders)
		if err != nil {
			return err
		}
		for i := range orders.Data {
			oType, err := order.StringToOrderType(orders.Data[i].OrderType)
			if err != nil {
				c.Websocket.DataHandler <- order.ClassificationError{
					Exchange: c.Name,
					OrderID:  orders.Data[i].OrderID,
					Err:      err,
				}
			}
			oStatus, err := order.StringToOrderStatus(orders.Data[i].Status)
			if err != nil {
				c.Websocket.DataHandler <- order.ClassificationError{
					Exchange: c.Name,
					OrderID:  orders.Data[i].OrderID,
					Err:      err,
				}
			}
			c.Websocket.DataHandler <- &order.Detail{
				Price:           orders.Data[i].OrderPrice,
				Amount:          orders.Data[i].Quantity,
				ExecutedAmount:  orders.Data[i].FilledQuantity,
				RemainingAmount: orders.Data[i].Quantity - orders.Data[i].FilledQuantity,
				Fee:             orders.Data[i].Fee,
				Exchange:        c.Name,
				ID:              orders.Data[i].OrderID,
				Type:            oType,
				Status:          oStatus,
				AssetType:       asset.PerpetualSwap,
				Date:            orders.Data[i].OrderTime,
				Leverage:        strconv.FormatInt(orders.Data[i].Leverage, 10),
				Pair: currency.NewPairFromFormattedPairs(orders.Data[i].Symbol,
					c.GetEnabledPairs(asset.PerpetualSwap),
					c.GetPairFormat(asset.PerpetualSwap, true)),
			}
		}
	default:
		c.Websocket.DataHandler <- wshandler.UnhandledMessageWarning{Message: c.Name + wshandler.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
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
