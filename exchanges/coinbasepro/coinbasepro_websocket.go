package coinbasepro

import (
	"context"
	"encoding/hex"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	coinbaseproWebsocketURL = "wss://advanced-trade-ws.coinbase.com"
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

	c.Websocket.Wg.Add(1)
	go c.wsReadData()
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (c *CoinbasePro) wsReadData() {
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

func (c *CoinbasePro) wsHandleData(respRaw []byte) error {
	fmt.Println("WHADDUP:", string(respRaw))
	genData := wsGen{}
	err := json.Unmarshal(respRaw, &genData)
	if err != nil {
		return err
	}

	if genData.Channel == "subscriptions" || genData.Channel == "heartbeats" {
		return nil
	}

	fmt.Printf("=== OH NOO LOOK AT THIS DATA WE HAVE TO DEAL WITH: %s ===\n", genData.Events)

	switch genData.Channel {
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
		var wsTicker []WebsocketTicker
		if len(genData.Events) == 0 {
			return errNoEventsWS
		}
		err := json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}

		for i := range wsTicker {
			c.Websocket.DataHandler <- &ticker.Price{
				LastUpdated:  genData.Timestamp,
				Pair:         wsTicker[i].ProductID,
				AssetType:    asset.Spot,
				ExchangeName: c.Name,
				High:         wsTicker[i].High24H,
				Low:          wsTicker[i].Low24H,
				Last:         wsTicker[i].Price,
				Volume:       wsTicker[i].Volume24H,
			}
		}
	case "snapshot":
		var snapshot WebsocketOrderbookSnapshot
		err := json.Unmarshal(respRaw, &snapshot)
		if err != nil {
			return err
		}

		err = c.ProcessSnapshot(&snapshot)
		if err != nil {
			return err
		}
	case "l2update":
		var update WebsocketL2Update
		err := json.Unmarshal(respRaw, &update)
		if err != nil {
			return err
		}

		err = c.ProcessUpdate(&update)
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

		creds, err := c.GetCredentials(context.TODO())
		if err != nil {
			c.Websocket.DataHandler <- order.ClassificationError{
				Exchange: c.Name,
				OrderID:  wsOrder.OrderID,
				Err:      err,
			}
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
				OrderID:         wsOrder.OrderID,
				AccountID:       wsOrder.ProfileID,
				ClientID:        creds.ClientID,
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
				OrderID:   wsOrder.OrderID,
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
	pair, err := currency.NewPairFromString(snapshot.ProductID)
	if err != nil {
		return err
	}

	base := orderbook.Base{
		Pair: pair,
		Bids: make(orderbook.Items, len(snapshot.Bids)),
		Asks: make(orderbook.Items, len(snapshot.Asks)),
	}

	for i := range snapshot.Bids {
		var price float64
		price, err = strconv.ParseFloat(snapshot.Bids[i][0], 64)
		if err != nil {
			return err
		}
		var amount float64
		amount, err = strconv.ParseFloat(snapshot.Bids[i][1], 64)
		if err != nil {
			return err
		}
		base.Bids[i] = orderbook.Item{Price: price, Amount: amount}
	}

	for i := range snapshot.Asks {
		var price float64
		price, err = strconv.ParseFloat(snapshot.Asks[i][0], 64)
		if err != nil {
			return err
		}
		var amount float64
		amount, err = strconv.ParseFloat(snapshot.Asks[i][1], 64)
		if err != nil {
			return err
		}
		base.Asks[i] = orderbook.Item{Price: price, Amount: amount}
	}

	base.Asset = asset.Spot
	base.Pair = pair
	base.Exchange = c.Name
	base.VerifyOrderbook = c.CanVerifyOrderbook
	base.LastUpdated = snapshot.Time
	return c.Websocket.Orderbook.LoadSnapshot(&base)
}

// ProcessUpdate updates the orderbook local cache
func (c *CoinbasePro) ProcessUpdate(update *WebsocketL2Update) error {
	if len(update.Changes) == 0 {
		return errors.New("no data in websocket update")
	}

	p, err := currency.NewPairFromString(update.ProductID)
	if err != nil {
		return err
	}

	asks := make(orderbook.Items, 0, len(update.Changes))
	bids := make(orderbook.Items, 0, len(update.Changes))

	for i := range update.Changes {
		price, err := strconv.ParseFloat(update.Changes[i][1], 64)
		if err != nil {
			return err
		}
		volume, err := strconv.ParseFloat(update.Changes[i][2], 64)
		if err != nil {
			return err
		}
		if update.Changes[i][0] == order.Buy.Lower() {
			bids = append(bids, orderbook.Item{Price: price, Amount: volume})
		} else {
			asks = append(asks, orderbook.Item{Price: price, Amount: volume})
		}
	}

	return c.Websocket.Orderbook.Update(&orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       p,
		UpdateTime: update.Time,
		Asset:      asset.Spot,
	})
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (c *CoinbasePro) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{
		"heartbeats",
		// "level2_batch", /*Other orderbook feeds require authentication. This is batched in 50ms lots.*/
		"ticker",
		// "user",
		// "matches",
	}
	enabledCurrencies, err := c.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions []stream.ChannelSubscription
	for i := range channels {
		for j := range enabledCurrencies {
			fPair, err := c.FormatExchangeCurrency(enabledCurrencies[j],
				asset.Spot)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channels[i],
				Currency: fPair,
				Asset:    asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

func (c *CoinbasePro) sendRequest(msgType, channel string, productID currency.Pair) error {
	creds, err := c.GetCredentials(context.Background())
	if err != nil {
		return err
	}

	n := strconv.FormatInt(time.Now().Unix(), 10)

	message := n + channel + productID.String()

	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(message),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}

	req := WebsocketSubscribe{
		Type:       msgType,
		ProductIDs: []string{productID.String()},
		Channel:    channel,
		Signature:  hex.EncodeToString(hmac),
		Key:        creds.Key,
		Timestamp:  n,
	}

	meow, _ := json.Marshal(req)

	fmt.Print(string(meow))

	err = c.Websocket.Conn.SendJSONMessage(req)
	if err != nil {
		return err
	}
	return nil
}

// Subscribe sends a websocket message to receive data from the channel
func (c *CoinbasePro) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {

	fmt.Printf("SUBSCRIBE: %v\n", channelsToSubscribe)
	// 	var creds *account.Credentials
	// 	var err error
	// 	if c.IsWebsocketAuthenticationSupported() {
	// 		creds, err = c.GetCredentials(context.TODO())
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}

	// 	subscribe := WebsocketSubscribe{
	// 		Type: "subscribe",
	// 	}

	// subscriptions:
	// 	for i := range channelsToSubscribe {
	// 		p := channelsToSubscribe[i].Currency.String()
	// 		if !common.StringDataCompare(subscribe.ProductIDs, p) && p != "" {
	// 			subscribe.ProductIDs = append(subscribe.ProductIDs, p)
	// 		}

	// 		if subscribe.Channel == channelsToSubscribe[i].Channel {
	// 			continue subscriptions
	// 		}

	// 		subscribe.Channel = channelsToSubscribe[i].Channel

	// 		if (channelsToSubscribe[i].Channel == "user" ||
	// 			channelsToSubscribe[i].Channel == "full") && creds != nil {
	// 			n := strconv.FormatInt(time.Now().Unix(), 10)
	// 			message := n + http.MethodGet + "/users/self/verify"
	// 			var hmac []byte
	// 			hmac, err = crypto.GetHMAC(crypto.HashSHA256,
	// 				[]byte(message),
	// 				[]byte(creds.Secret))
	// 			if err != nil {
	// 				return err
	// 			}
	// 			subscribe.Signature = crypto.Base64Encode(hmac)
	// 			subscribe.Key = creds.Key
	// 			subscribe.Timestamp = n
	// 		}
	// 	}
	for i := range channelsToSubscribe {
		err := c.sendRequest("subscribe", channelsToSubscribe[i].Channel, channelsToSubscribe[i].Currency)
		if err != nil {
			return err
		}
	}

	c.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (c *CoinbasePro) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	unsubscribe := WebsocketSubscribe{
		Type: "unsubscribe",
	}

	for i := range channelsToUnsubscribe {
		p := channelsToUnsubscribe[i].Currency.String()
		if !common.StringDataCompare(unsubscribe.ProductIDs, p) && p != "" {
			unsubscribe.ProductIDs = append(unsubscribe.ProductIDs, p)
		}

		if unsubscribe.Channel == channelsToUnsubscribe[i].Channel {
			unsubscribe.Channel = channelsToUnsubscribe[i].Channel

		}

	}
	err := c.Websocket.Conn.SendJSONMessage(unsubscribe)
	if err != nil {
		return err
	}
	c.Websocket.RemoveSubscriptions(channelsToUnsubscribe...)
	return nil
}

// const wow = "-----BEGIN EC PRIVATE KEY-----\n%s\n-----END EC PRIVATE KEY-----\n"

// func (c *CoinbasePro) GetJWT(ctx context.Context) (string, error) {
// 	creds, err := c.GetCredentials(ctx)
// 	if err != nil {
// 		return "", err
// 	}

// 	block, _ := pem.Decode([]byte(fmt.Sprintf(wow, creds.Secret)))
//     if block == nil {
//         return "", fmt.Errorf("jwt: Could not decode private key")
//     }

// }
