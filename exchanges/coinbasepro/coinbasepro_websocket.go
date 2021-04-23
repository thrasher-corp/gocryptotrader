package coinbasepro

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
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
	coinbaseproWebsocketURL = "wss://ws-feed.pro.coinbase.com"
)

// WsConnect initiates a websocket connection
func (c *CoinbasePro) WsConnect() error {
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := c.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	go c.wsReadData()
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (c *CoinbasePro) wsReadData() {
	c.Websocket.Wg.Add(1)

	defer func() {
		c.Websocket.Wg.Done()
	}()

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

func (c *CoinbasePro) wsHandleData(respRaw []byte) error {
	msgType := wsMsgType{}
	err := json.Unmarshal(respRaw, &msgType)
	if err != nil {
		return err
	}

	if msgType.Type == "subscriptions" || msgType.Type == "heartbeat" {
		return nil
	}

	switch msgType.Type {
	case "status":
		var status wsStatus
		err = json.Unmarshal(respRaw, &status)
		if err != nil {
			return err
		}
		c.Websocket.DataHandler <- status
	case "error":
		c.Websocket.DataHandler <- errors.New(string(respRaw))
	case "ticker":
		wsTicker := WebsocketTicker{}
		err := json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}

		c.Websocket.DataHandler <- &ticker.Price{
			LastUpdated:  wsTicker.Time,
			Pair:         wsTicker.ProductID,
			AssetType:    asset.Spot,
			ExchangeName: c.Name,
			Open:         wsTicker.Open24H,
			High:         wsTicker.High24H,
			Low:          wsTicker.Low24H,
			Last:         wsTicker.Price,
			Volume:       wsTicker.Volume24H,
			Bid:          wsTicker.BestBid,
			Ask:          wsTicker.BestAsk,
		}

	case "snapshot":
		snapshot := WebsocketOrderbookSnapshot{}
		err := json.Unmarshal(respRaw, &snapshot)
		if err != nil {
			return err
		}

		err = c.ProcessSnapshot(&snapshot)
		if err != nil {
			return err
		}

	case "l2update":
		update := WebsocketL2Update{}
		err := json.Unmarshal(respRaw, &update)
		if err != nil {
			return err
		}

		err = c.ProcessUpdate(update)
		if err != nil {
			return err
		}
	case "received", "open", "done", "change", "activate":
		var wsOrder wsOrderReceived
		err := json.Unmarshal(respRaw, &wsOrder)
		if err != nil {
			return err
		}
		var oType order.Type
		var oSide order.Side
		var oStatus order.Status
		oType, err = order.StringToOrderType(wsOrder.OrderType)
		if err != nil {
			c.Websocket.DataHandler <- order.ClassificationError{
				Exchange: c.Name,
				OrderID:  wsOrder.OrderID,
				Err:      err,
			}
		}
		oSide, err = order.StringToOrderSide(wsOrder.Side)
		if err != nil {
			c.Websocket.DataHandler <- order.ClassificationError{
				Exchange: c.Name,
				OrderID:  wsOrder.OrderID,
				Err:      err,
			}
		}
		oStatus, err = statusToStandardStatus(wsOrder.Type)
		if err != nil {
			c.Websocket.DataHandler <- order.ClassificationError{
				Exchange: c.Name,
				OrderID:  wsOrder.OrderID,
				Err:      err,
			}
		}
		if wsOrder.Reason == "canceled" {
			oStatus = order.Cancelled
		}
		ts := wsOrder.Time
		if wsOrder.Type == "activate" {
			ts = convert.TimeFromUnixTimestampDecimal(wsOrder.Timestamp)
		}

		if wsOrder.UserID != "" {
			var p currency.Pair
			var a asset.Item
			p, a, err = c.GetRequestFormattedPairAndAssetType(wsOrder.ProductID)
			if err != nil {
				return err
			}
			c.Websocket.DataHandler <- &order.Detail{
				HiddenOrder:     wsOrder.Private,
				Price:           wsOrder.Price,
				Amount:          wsOrder.Size,
				TriggerPrice:    wsOrder.StopPrice,
				ExecutedAmount:  wsOrder.Size - wsOrder.RemainingSize,
				RemainingAmount: wsOrder.RemainingSize,
				Fee:             wsOrder.TakerFeeRate,
				Exchange:        c.Name,
				ID:              wsOrder.OrderID,
				AccountID:       wsOrder.ProfileID,
				ClientID:        c.API.Credentials.ClientID,
				Type:            oType,
				Side:            oSide,
				Status:          oStatus,
				AssetType:       a,
				Date:            ts,
				Pair:            p,
			}
		}
	case "match":
		var wsOrder wsOrderReceived
		err := json.Unmarshal(respRaw, &wsOrder)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(wsOrder.Side)
		if err != nil {
			c.Websocket.DataHandler <- order.ClassificationError{
				Exchange: c.Name,
				Err:      err,
			}
		}
		var p currency.Pair
		var a asset.Item
		p, a, err = c.GetRequestFormattedPairAndAssetType(wsOrder.ProductID)
		if err != nil {
			return err
		}

		if wsOrder.UserID != "" {
			c.Websocket.DataHandler <- &order.Detail{
				ID:        wsOrder.OrderID,
				Pair:      p,
				AssetType: a,
				Trades: []order.TradeHistory{
					{
						Price:     wsOrder.Price,
						Amount:    wsOrder.Size,
						Exchange:  c.Name,
						TID:       strconv.FormatInt(wsOrder.TradeID, 10),
						Side:      oSide,
						Timestamp: wsOrder.Time,
						IsMaker:   wsOrder.TakerUserID == "",
					},
				},
			}
		} else {
			if !c.IsSaveTradeDataEnabled() {
				return nil
			}
			return trade.AddTradesToBuffer(c.Name, trade.Data{
				Timestamp:    wsOrder.Time,
				Exchange:     c.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        wsOrder.Price,
				Amount:       wsOrder.Size,
				Side:         oSide,
				TID:          strconv.FormatInt(wsOrder.TradeID, 10),
			})
		}
	default:
		c.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: c.Name + stream.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
}

func statusToStandardStatus(stat string) (order.Status, error) {
	switch stat {
	case "received":
		return order.New, nil
	case "open":
		return order.Active, nil
	case "done":
		return order.Filled, nil
	case "match":
		return order.PartiallyFilled, nil
	case "change", "activate":
		return order.Active, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%s not recognised as status type", stat)
	}
}

// ProcessSnapshot processes the initial orderbook snap shot
func (c *CoinbasePro) ProcessSnapshot(snapshot *WebsocketOrderbookSnapshot) error {
	var base orderbook.Base
	for i := range snapshot.Bids {
		price, err := strconv.ParseFloat(snapshot.Bids[i][0].(string), 64)
		if err != nil {
			return err
		}

		amount, err := strconv.ParseFloat(snapshot.Bids[i][1].(string), 64)
		if err != nil {
			return err
		}

		base.Bids = append(base.Bids,
			orderbook.Item{Price: price, Amount: amount})
	}

	for i := range snapshot.Asks {
		price, err := strconv.ParseFloat(snapshot.Asks[i][0].(string), 64)
		if err != nil {
			return err
		}

		amount, err := strconv.ParseFloat(snapshot.Asks[i][1].(string), 64)
		if err != nil {
			return err
		}

		base.Asks = append(base.Asks,
			orderbook.Item{Price: price, Amount: amount})
	}

	pair, err := currency.NewPairFromString(snapshot.ProductID)
	if err != nil {
		return err
	}

	base.Asset = asset.Spot
	base.Pair = pair
	base.Exchange = c.Name
	base.VerifyOrderbook = c.CanVerifyOrderbook

	return c.Websocket.Orderbook.LoadSnapshot(&base)
}

// ProcessUpdate updates the orderbook local cache
func (c *CoinbasePro) ProcessUpdate(update WebsocketL2Update) error {
	var asks, bids []orderbook.Item

	for i := range update.Changes {
		price, err := strconv.ParseFloat(update.Changes[i][1].(string), 64)
		if err != nil {
			return err
		}
		volume, err := strconv.ParseFloat(update.Changes[i][2].(string), 64)
		if err != nil {
			return err
		}

		if update.Changes[i][0].(string) == order.Buy.Lower() {
			bids = append(bids, orderbook.Item{Price: price, Amount: volume})
		} else {
			asks = append(asks, orderbook.Item{Price: price, Amount: volume})
		}
	}

	if len(asks) == 0 && len(bids) == 0 {
		return errors.New("no data in websocket update")
	}

	p, err := currency.NewPairFromString(update.ProductID)
	if err != nil {
		return err
	}

	timestamp, err := time.Parse(time.RFC3339, update.Time)
	if err != nil {
		return err
	}
	return c.Websocket.Orderbook.Update(&buffer.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       p,
		UpdateTime: timestamp,
		Asset:      asset.Spot,
	})
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (c *CoinbasePro) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{"heartbeat", "level2", "ticker", "user", "matches"}
	enabledCurrencies, err := c.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions []stream.ChannelSubscription
	for i := range channels {
		if (channels[i] == "user" || channels[i] == "full") &&
			!c.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
			continue
		}
		for j := range enabledCurrencies {
			fpair, err := c.FormatExchangeCurrency(enabledCurrencies[j],
				asset.Spot)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channels[i],
				Currency: fpair,
				Asset:    asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (c *CoinbasePro) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	subscribe := WebsocketSubscribe{
		Type: "subscribe",
	}

subscriptions:
	for i := range channelsToSubscribe {
		p := channelsToSubscribe[i].Currency.String()
		if !common.StringDataCompare(subscribe.ProductIDs, p) && p != "" {
			subscribe.ProductIDs = append(subscribe.ProductIDs, p)
		}

		for j := range subscribe.Channels {
			if subscribe.Channels[j].Name == channelsToSubscribe[i].Channel {
				continue subscriptions
			}
		}

		subscribe.Channels = append(subscribe.Channels, WsChannels{
			Name: channelsToSubscribe[i].Channel,
		})

		if channelsToSubscribe[i].Channel == "user" ||
			channelsToSubscribe[i].Channel == "full" {
			n := strconv.FormatInt(time.Now().Unix(), 10)
			message := n + http.MethodGet + "/users/self/verify"
			hmac := crypto.GetHMAC(crypto.HashSHA256, []byte(message),
				[]byte(c.API.Credentials.Secret))
			subscribe.Signature = crypto.Base64Encode(hmac)
			subscribe.Key = c.API.Credentials.Key
			subscribe.Passphrase = c.API.Credentials.ClientID
			subscribe.Timestamp = n
		}
	}
	err := c.Websocket.Conn.SendJSONMessage(subscribe)
	if err != nil {
		return err
	}
	c.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (c *CoinbasePro) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	unsubscribe := WebsocketSubscribe{
		Type: "unsubscribe",
	}

unsubscriptions:
	for i := range channelsToUnsubscribe {
		p := channelsToUnsubscribe[i].Currency.String()
		if !common.StringDataCompare(unsubscribe.ProductIDs, p) && p != "" {
			unsubscribe.ProductIDs = append(unsubscribe.ProductIDs, p)
		}

		for j := range unsubscribe.Channels {
			if unsubscribe.Channels[j].Name == channelsToUnsubscribe[i].Channel {
				continue unsubscriptions
			}
		}

		unsubscribe.Channels = append(unsubscribe.Channels, WsChannels{
			Name: channelsToUnsubscribe[i].Channel,
		})
	}
	err := c.Websocket.Conn.SendJSONMessage(unsubscribe)
	if err != nil {
		return err
	}
	c.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe...)
	return nil
}
