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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	wsContractURL = "wss://ws.coinbene.com/stream/ws"
	event         = "event"
	topic         = "topic"
)

// WsConnect connects to websocket
func (c *Coinbene) WsConnect() error {
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := c.Websocket.Conn.Dial(&dialer, http.Header{})
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
	subs, err := c.GenerateDefaultSubscriptions()
	if err != nil {
		return err
	}
	return c.Websocket.SubscribeToChannels(subs)
}

// GenerateDefaultSubscriptions generates stuff
func (c *Coinbene) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{"orderBook.%s.100", "tradeList.%s", "ticker.%s", "kline.%s.1m"}
	var subscriptions []stream.ChannelSubscription
	pairs, err := c.GetEnabledPairs(asset.PerpetualSwap)
	if err != nil {
		return nil, err
	}
	for x := range channels {
		for y := range pairs {
			pairs[y].Delimiter = ""
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  "btc/" + fmt.Sprintf(channels[x], pairs[y]),
				Currency: pairs[y],
				Asset:    asset.PerpetualSwap,
			})
		}
	}

	pairs, err = c.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for x := range channels {
		for y := range pairs {
			pairs[y].Delimiter = ""
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  "spot/" + fmt.Sprintf(channels[x], pairs[y]),
				Currency: pairs[y],
				Asset:    asset.Spot,
			})
		}
	}

	return subscriptions, nil
}

// GenerateAuthSubs generates auth subs
func (c *Coinbene) GenerateAuthSubs() ([]stream.ChannelSubscription, error) {
	var subscriptions []stream.ChannelSubscription
	var sub stream.ChannelSubscription
	var userChannels = []string{"user.account", "user.position", "user.order"}
	for z := range userChannels {
		sub.Channel = userChannels[z]
		subscriptions = append(subscriptions, sub)
	}
	return subscriptions, nil
}

// wsReadData receives and passes on websocket messages for processing
func (c *Coinbene) wsReadData() {
	c.Websocket.Wg.Add(1)
	defer c.Websocket.Wg.Done()
	for {
		resp := c.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := c.wsHandleData(resp.Raw)
		if err != nil {
			c.Websocket.DataHandler <- err
		}
	}
}

func inferAssetFromTopic(topic string) asset.Item {
	switch {
	case strings.Contains(topic, "spot/"):
		return asset.Spot
	default:
		return asset.PerpetualSwap
	}
}

func (c *Coinbene) wsHandleData(respRaw []byte) error {
	if string(respRaw) == stream.Ping {
		err := c.Websocket.Conn.SendRawMessage(websocket.TextMessage, []byte(stream.Pong))
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
			var authsubs []stream.ChannelSubscription
			authsubs, err = c.GenerateAuthSubs()
			if err != nil {
				return err
			}
			return c.Websocket.SubscribeToChannels(authsubs)
		}
		c.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return fmt.Errorf("message: %s. code: %v", result["message"], result["code"])
	}
	assetType := inferAssetFromTopic(result[topic].(string))
	switch {
	case strings.Contains(result[topic].(string), "ticker"):
		var wsTicker WsTicker
		err = json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}

		var format currency.PairFormat
		format, err = c.GetPairFormat(assetType, true)
		if err != nil {
			return err
		}

		var pairs currency.Pairs
		pairs, err = c.GetEnabledPairs(assetType)
		if err != nil {
			return err
		}

		for x := range wsTicker.Data {
			var p currency.Pair
			p, err = currency.NewPairFromFormattedPairs(wsTicker.Data[x].Symbol,
				pairs,
				format)
			if err != nil {
				return err
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
				AssetType:   assetType,
				LastUpdated:  time.Unix(wsTicker.Data[x].Timestamp, 0),
			}
		}
	case strings.Contains(result[topic].(string), "tradeList"):
		var tradeList WsTradeList
		err = json.Unmarshal(respRaw, &tradeList)
		if err != nil {
			return err
		}
		var trades []trade.Data
		for i := range tradeList.Data {
			var price, amount float64
			t := time.Unix(int64(tradeList.Data[i][3].(float64)), 0)
			price, err = strconv.ParseFloat(tradeList.Data[i][0].(string), 64)
			if err != nil {
				return err
			}
			amount, err = strconv.ParseFloat(tradeList.Data[i][2].(string), 64)
			if err != nil {
				return err
			}

			var tSide = order.Buy
			if tradeList.Data[i][1] == "s" {
				tSide = order.Sell
			}

			var format currency.PairFormat
			format, err = c.GetPairFormat(assetType, true)
			if err != nil {
				return err
			}

			var pairs currency.Pairs
			pairs, err = c.GetEnabledPairs(assetType)
			if err != nil {
				return err
			}

			var newP currency.Pair
			p := strings.Split(tradeList.Topic, ".")
			if len(p) == 1 {
				return errors.New("something")
			}
			newP, err = currency.NewPairFromFormattedPairs(p[1], pairs, format)
			if err != nil {
				return err
			}

			trades = append(trades, trade.Data{
				Timestamp:    t,
				Exchange:     c.Name,
				CurrencyPair: newP,
				AssetType:    assetType,
				Price:        price,
				Amount:       amount,
				Side:         tSide,
			})
		}
		c.Websocket.Trade.AddTradesToBuffer(trades...)
	case strings.Contains(result[topic].(string), "orderBook"):
		var orderBook WsOrderbookData
		err = json.Unmarshal(respRaw, &orderBook)
		if err != nil {
			return err
		}

		var format currency.PairFormat
		format, err = c.GetPairFormat(assetType, true)
		if err != nil {
			return err
		}

		var pairs currency.Pairs
		pairs, err = c.GetEnabledPairs(assetType)
		if err != nil {
			return err
		}
		p := strings.Split(orderBook.Topic, ".")
		if len(p) == 1 {
			return errors.New("crikey m8")
		}
		var newP currency.Pair
		newP, err = currency.NewPairFromFormattedPairs(p[1], pairs, format)
		if err != nil {
			return err
		}
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
			newOB.AssetType = assetType
			newOB.Pair = newP
			newOB.ExchangeName = c.Name
			newOB.LastUpdated = time.Unix(orderBook.Data[0].Timestamp, 0)
			err = c.Websocket.Orderbook.LoadSnapshot(&newOB)
			if err != nil {
				return err
			}
		} else if orderBook.Action == "update" {
			newOB := buffer.Update{
				Asks:       asks,
				Bids:       bids,
				Asset:      assetType,
				Pair:       newP,
				UpdateID:   orderBook.Data[0].Version,
				UpdateTime: time.Unix(orderBook.Data[0].Timestamp,0),
			}
			err = c.Websocket.Orderbook.Update(&newOB)
			if err != nil {
				return err
			}
		}
	case strings.Contains(result[topic].(string), "kline"):
		var kline WsKline
		err = json.Unmarshal(respRaw, &kline)
		if err != nil {
			return err
		}
		var format currency.PairFormat
		format, err = c.GetPairFormat(assetType, true)
		if err != nil {
			return err
		}

		var pairs currency.Pairs
		pairs, err = c.GetEnabledPairs(assetType)
		if err != nil {
			return err
		}

		var newP currency.Pair
		p := strings.Split(kline.Topic, ".")
		if len(p) == 1 {
			return errors.New("crikey m8")
		}
		newP, err = currency.NewPairFromFormattedPairs(p[1], pairs, format)
		if err != nil {
			return err
		}

		for i := range kline.Data {
			c.Websocket.DataHandler <- stream.KlineData{
				Timestamp:  time.Unix(kline.Data[i].T, 0),
				Pair:       newP,
				AssetType:  assetType,
				Exchange:   c.Name,
				OpenPrice:  kline.Data[i].O,
				ClosePrice: kline.Data[i].C,
				HighPrice:  kline.Data[i].H,
				LowPrice:   kline.Data[i].L,
				Volume:     kline.Data[i].V,
			}
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

		format, err := c.GetPairFormat(assetType, true)
		if err != nil {
			return err
		}

		pairs, err := c.GetEnabledPairs(assetType)
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

			newP, err := currency.NewPairFromFormattedPairs(orders.Data[i].Symbol,
				pairs,
				format)
			if err != nil {
				return err
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
				AssetType:       assetType,
				Date:            time.Unix(orders.Data[i].OrderTime, 0),
				Leverage:        strconv.FormatInt(orders.Data[i].Leverage, 10),
				Pair:            newP,
			}
		}
	default:
		c.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: c.Name + stream.UnhandledMessage + string(respRaw),
		}
		return nil
	}
	return nil
}

// Subscribe sends a websocket message to receive data from the channel
func (c *Coinbene) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	var sub WsSub
	sub.Operation = "subscribe"
	charLim := 1
	for i := range channelsToSubscribe {
		if len(sub.Arguments) > charLim {
			err := c.Websocket.Conn.SendJSONMessage(sub)
			if err != nil {
				return err
			}
			sub.Arguments = []string{}
		}
		sub.Arguments = append(sub.Arguments, channelsToSubscribe[i].Channel)
	}
	err := c.Websocket.Conn.SendJSONMessage(sub)
	if err != nil {
		return err
	}
	c.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}

// Unsubscribe sends a websocket message to receive data from the channel
func (c *Coinbene) Unsubscribe(channelToUnsubscribe []stream.ChannelSubscription) error {
	var unsub WsSub
	unsub.Operation = "unsubscribe"
	for i := range channelToUnsubscribe {
		unsub.Arguments = append(unsub.Arguments, channelToUnsubscribe[i].Channel)
	}
	err := c.Websocket.Conn.SendJSONMessage(unsub)
	if err != nil {
		return err
	}
	c.Websocket.RemoveSuccessfulUnsubscriptions(channelToUnsubscribe...)
	return nil
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
	return c.Websocket.Conn.SendJSONMessage(sub)
}
