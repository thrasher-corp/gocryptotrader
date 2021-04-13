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
	wsContractURL     = "wss://ws.coinbene.com/stream/ws"
	event             = "event"
	topic             = "topic"
	swapChannelPrefix = "btc/"
	spotChannelPrefix = "spot/"
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
	return nil
}

// GenerateDefaultSubscriptions generates stuff
func (c *Coinbene) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{"orderBook.%s.100", "tradeList.%s", "ticker.%s", "kline.%s.1h"}
	var subscriptions []stream.ChannelSubscription
	perpetualPairs, err := c.GetEnabledPairs(asset.PerpetualSwap)
	if err != nil {
		return nil, err
	}
	var spotPairs currency.Pairs
	spotPairs, err = c.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for x := range channels {
		for y := range perpetualPairs {
			perpetualPairs[y].Delimiter = ""
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  swapChannelPrefix + fmt.Sprintf(channels[x], perpetualPairs[y]),
				Currency: perpetualPairs[y],
				Asset:    asset.PerpetualSwap,
			})
		}
		for z := range spotPairs {
			spotPairs[z].Delimiter = ""
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  spotChannelPrefix + fmt.Sprintf(channels[x], spotPairs[z]),
				Currency: spotPairs[z],
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
	if strings.Contains(topic, "spot/") {
		return asset.Spot
	}
	return asset.PerpetualSwap
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
	var newPair currency.Pair
	switch {
	case strings.Contains(result[topic].(string), "ticker"):
		var wsTicker WsTicker
		err = json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}
		newPair, err = c.getCurrencyFromWsTopic(assetType, wsTicker.Topic)
		if err != nil {
			return err
		}

		for x := range wsTicker.Data {
			c.Websocket.DataHandler <- &ticker.Price{
				Volume:       wsTicker.Data[x].Volume24h,
				Last:         wsTicker.Data[x].LastPrice,
				High:         wsTicker.Data[x].High24h,
				Low:          wsTicker.Data[x].Low24h,
				Bid:          wsTicker.Data[x].BestBidPrice,
				Ask:          wsTicker.Data[x].BestAskPrice,
				Pair:         newPair,
				ExchangeName: c.Name,
				AssetType:    assetType,
				LastUpdated:  time.Unix(wsTicker.Data[x].Timestamp, 0),
			}
		}
	case strings.Contains(result[topic].(string), "tradeList"):
		if !c.IsSaveTradeDataEnabled() {
			return nil
		}
		var tradeList WsTradeList
		err = json.Unmarshal(respRaw, &tradeList)
		if err != nil {
			return err
		}
		var trades []trade.Data
		for i := range tradeList.Data {
			var price, amount float64
			t := time.Unix(int64(tradeList.Data[i][3].(float64))/1000, 0)
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

			newPair, err = c.getCurrencyFromWsTopic(assetType, tradeList.Topic)
			if err != nil {
				return err
			}

			trades = append(trades, trade.Data{
				Timestamp:    t,
				Exchange:     c.Name,
				CurrencyPair: newPair,
				AssetType:    assetType,
				Price:        price,
				Amount:       amount,
				Side:         tSide,
			})
		}
		return trade.AddTradesToBuffer(c.Name, trades...)
	case strings.Contains(result[topic].(string), "orderBook"):
		var orderBook WsOrderbookData
		err = json.Unmarshal(respRaw, &orderBook)
		if err != nil {
			return err
		}

		if len(orderBook.Data) != 1 {
			return errors.New("incomplete orderbook data has been received")
		}

		newPair, err = c.getCurrencyFromWsTopic(assetType, orderBook.Topic)
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
			price, err = strconv.ParseFloat(orderBook.Data[0].Bids[j][0], 64)
			if err != nil {
				return err
			}

			if price == 0 {
				// Last level is coming back as a float with not enough decimal
				// places e.g. ["0.000","1001.95"]],
				// This needs to be filtered out as this can skew orderbook
				// calculations
				continue
			}

			amount, err = strconv.ParseFloat(orderBook.Data[0].Bids[j][1], 64)
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
			newOB.Asset = assetType
			newOB.Pair = newPair
			newOB.Exchange = c.Name
			newOB.LastUpdated = time.Unix(orderBook.Data[0].Timestamp, 0)
			newOB.VerifyOrderbook = c.CanVerifyOrderbook
			err = c.Websocket.Orderbook.LoadSnapshot(&newOB)
			if err != nil {
				return err
			}
		} else if orderBook.Action == "update" {
			newOB := buffer.Update{
				Asks:       asks,
				Bids:       bids,
				Asset:      assetType,
				Pair:       newPair,
				UpdateID:   orderBook.Data[0].Version,
				UpdateTime: time.Unix(orderBook.Data[0].Timestamp, 0),
			}
			err = c.Websocket.Orderbook.Update(&newOB)
			if err != nil {
				return err
			}
		}
	case strings.Contains(result[topic].(string), "kline"):
		var candleData WsKline
		err = json.Unmarshal(respRaw, &candleData)
		if err != nil {
			return err
		}
		newPair, err = c.getCurrencyFromWsTopic(assetType, candleData.Topic)
		if err != nil {
			return err
		}

		for i := range candleData.Data {
			c.Websocket.DataHandler <- stream.KlineData{
				Pair:       newPair,
				AssetType:  assetType,
				Exchange:   c.Name,
				OpenPrice:  candleData.Data[i].Open,
				HighPrice:  candleData.Data[i].High,
				LowPrice:   candleData.Data[i].Low,
				ClosePrice: candleData.Data[i].Close,
				Volume:     candleData.Data[i].Volume,
				Timestamp:  time.Unix(candleData.Data[i].Timestamp, 0),
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

			newPair, err = currency.NewPairFromFormattedPairs(orders.Data[i].Symbol,
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
				Date:            orders.Data[i].OrderTime,
				Leverage:        float64(orders.Data[i].Leverage),
				Pair:            newPair,
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

func (c *Coinbene) getCurrencyFromWsTopic(assetType asset.Item, channelTopic string) (cp currency.Pair, err error) {
	var format currency.PairFormat
	format, err = c.GetPairFormat(assetType, true)
	if err != nil {
		return cp, err
	}

	var pairs currency.Pairs
	pairs, err = c.GetEnabledPairs(assetType)
	if err != nil {
		return cp, err
	}
	// channel topics are formatted as "spot/orderbook.BTCUSDT"
	channelSplit := strings.Split(channelTopic, ".")
	if len(channelSplit) == 1 {
		return currency.Pair{}, errors.New("no currency found in topic " + channelTopic)
	}
	cp, err = currency.MatchPairsWithNoDelimiter(channelSplit[1], pairs, format)
	if err != nil {
		return cp, err
	}
	if !pairs.Contains(cp, true) {
		return cp, fmt.Errorf("currency %s not found in enabled pairs", cp.String())
	}
	return cp, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (c *Coinbene) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	maxSubsPerHour := 240
	if len(channelsToSubscribe) > maxSubsPerHour {
		return fmt.Errorf("channel subscriptions length %d exceeds coinbene's limit of %d, try reducing enabled pairs",
			len(channelsToSubscribe),
			maxSubsPerHour)
	}

	var sub WsSub
	sub.Operation = "subscribe"
	// enabling all currencies can lead to a message too large being sent
	// and no subscriptions being made
	chanLimit := 15
	for i := range channelsToSubscribe {
		if len(sub.Arguments) > chanLimit {
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
	// enabling all currencies can lead to a message too large being sent
	// and no unsubscribes being made
	chanLimit := 15
	for i := range channelToUnsubscribe {
		if len(unsub.Arguments) > chanLimit {
			err := c.Websocket.Conn.SendJSONMessage(unsub)
			if err != nil {
				return err
			}
			unsub.Arguments = []string{}
		}
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
