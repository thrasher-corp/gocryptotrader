package btse

import (
	"context"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	btseWebsocket      = "wss://ws.btse.com/ws/spot"
	btseWebsocketTimer = time.Second * 57
)

// WsConnect connects the websocket client
func (b *BTSE) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	err := b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	b.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		MessageType: websocket.PingMessage,
		Delay:       btseWebsocketTimer,
	})

	b.Websocket.Wg.Add(1)
	go b.wsReadData()

	if b.IsWebsocketAuthenticationSupported() {
		err = b.WsAuthenticate(context.TODO())
		if err != nil {
			b.Websocket.DataHandler <- err
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	return nil
}

// WsAuthenticate Send an authentication message to receive auth data
func (b *BTSE) WsAuthenticate(ctx context.Context) error {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}
	nonce := strconv.FormatInt(time.Now().UnixMilli(), 10)
	path := "/spotWS" + nonce

	hmac, err := crypto.GetHMAC(crypto.HashSHA512_384,
		[]byte((path)),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}

	sign := crypto.HexEncodeToString(hmac)
	req := wsSub{
		Operation: "authKeyExpires",
		Arguments: []string{creds.Key, nonce, sign},
	}
	return b.Websocket.Conn.SendJSONMessage(req)
}

func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "ORDER_INSERTED", "TRIGGER_INSERTED":
		return order.New, nil
	case "ORDER_CANCELLED":
		return order.Cancelled, nil
	case "ORDER_FULL_TRANSACTED":
		return order.Filled, nil
	case "ORDER_PARTIALLY_TRANSACTED":
		return order.PartiallyFilled, nil
	case "TRIGGER_ACTIVATED":
		return order.Active, nil
	case "INSUFFICIENT_BALANCE":
		return order.InsufficientBalance, nil
	case "MARKET_UNAVAILABLE":
		return order.MarketUnavailable, nil
	default:
		return order.UnknownStatus, errors.New(status + " not recognised as order status")
	}
}

// wsReadData receives and passes on websocket messages for processing
func (b *BTSE) wsReadData() {
	defer b.Websocket.Wg.Done()

	for {
		resp := b.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := b.wsHandleData(resp.Raw)
		if err != nil {
			b.Websocket.DataHandler <- err
		}
	}
}

func (b *BTSE) wsHandleData(respRaw []byte) error {
	type Result map[string]interface{}
	var result Result
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		if strings.Contains(string(respRaw), "connect success") {
			return nil
		}
		return err
	}
	if result == nil {
		return nil
	}

	if result["event"] != nil {
		event, ok := result["event"].(string)
		if !ok {
			return errors.New(b.Name + stream.UnhandledMessage + string(respRaw))
		}
		switch event {
		case "subscribe":
			var subscribe WsSubscriptionAcknowledgement
			err = json.Unmarshal(respRaw, &subscribe)
			if err != nil {
				return err
			}
			if b.Verbose {
				log.Infof(log.WebsocketMgr, "%v subscribed to %v", b.Name, strings.Join(subscribe.Channel, ", "))
			}
		case "login":
			var login WsLoginAcknowledgement
			err = json.Unmarshal(respRaw, &login)
			if err != nil {
				return err
			}
			b.Websocket.SetCanUseAuthenticatedEndpoints(login.Success)
			if b.Verbose {
				log.Infof(log.WebsocketMgr, "%v websocket authenticated: %v", b.Name, login.Success)
			}
		default:
			return errors.New(b.Name + stream.UnhandledMessage + string(respRaw))
		}
		return nil
	}

	topic, ok := result["topic"].(string)
	if !ok {
		return errors.New(b.Name + stream.UnhandledMessage + string(respRaw))
	}
	switch {
	case topic == "notificationApi":
		var notification wsNotification
		err = json.Unmarshal(respRaw, &notification)
		if err != nil {
			return err
		}
		for i := range notification.Data {
			var oType order.Type
			var oSide order.Side
			var oStatus order.Status
			oType, err = order.StringToOrderType(notification.Data[i].Type)
			if err != nil {
				b.Websocket.DataHandler <- order.ClassificationError{
					Exchange: b.Name,
					OrderID:  notification.Data[i].OrderID,
					Err:      err,
				}
			}
			oSide, err = order.StringToOrderSide(notification.Data[i].OrderMode)
			if err != nil {
				b.Websocket.DataHandler <- order.ClassificationError{
					Exchange: b.Name,
					OrderID:  notification.Data[i].OrderID,
					Err:      err,
				}
			}
			oStatus, err = stringToOrderStatus(notification.Data[i].Status)
			if err != nil {
				b.Websocket.DataHandler <- order.ClassificationError{
					Exchange: b.Name,
					OrderID:  notification.Data[i].OrderID,
					Err:      err,
				}
			}

			var p currency.Pair
			p, err = currency.NewPairFromString(notification.Data[i].Symbol)
			if err != nil {
				return err
			}

			var a asset.Item
			a, err = b.GetPairAssetType(p)
			if err != nil {
				return err
			}

			b.Websocket.DataHandler <- &order.Detail{
				Price:        notification.Data[i].Price,
				Amount:       notification.Data[i].Size,
				TriggerPrice: notification.Data[i].TriggerPrice,
				Exchange:     b.Name,
				OrderID:      notification.Data[i].OrderID,
				Type:         oType,
				Side:         oSide,
				Status:       oStatus,
				AssetType:    a,
				Date:         time.UnixMilli(notification.Data[i].Timestamp),
				Pair:         p,
			}
		}
	case strings.Contains(topic, "tradeHistory"):
		if !b.IsSaveTradeDataEnabled() {
			return nil
		}
		var tradeHistory wsTradeHistory
		err = json.Unmarshal(respRaw, &tradeHistory)
		if err != nil {
			return err
		}
		var trades []trade.Data
		for x := range tradeHistory.Data {
			side := order.Buy
			if tradeHistory.Data[x].Gain == -1 {
				side = order.Sell
			}

			var p currency.Pair
			p, err = currency.NewPairFromString(strings.Replace(tradeHistory.Topic,
				"tradeHistory:",
				"",
				1))
			if err != nil {
				return err
			}
			var a asset.Item
			a, err = b.GetPairAssetType(p)
			if err != nil {
				return err
			}
			trades = append(trades, trade.Data{
				Timestamp:    time.UnixMilli(tradeHistory.Data[x].TransactionTime),
				CurrencyPair: p,
				AssetType:    a,
				Exchange:     b.Name,
				Price:        tradeHistory.Data[x].Price,
				Amount:       tradeHistory.Data[x].Amount,
				Side:         side,
				TID:          strconv.FormatInt(tradeHistory.Data[x].ID, 10),
			})
		}
		return trade.AddTradesToBuffer(b.Name, trades...)
	case strings.Contains(topic, "orderBookL2Api"): // TODO: Fix orderbook updates.
		var t wsOrderBook
		err = json.Unmarshal(respRaw, &t)
		if err != nil {
			return err
		}
		newOB := orderbook.Base{
			Bids: make(orderbook.Tranches, 0, len(t.Data.BuyQuote)),
			Asks: make(orderbook.Tranches, 0, len(t.Data.SellQuote)),
		}
		var price, amount float64
		for i := range t.Data.SellQuote {
			p := strings.Replace(t.Data.SellQuote[i].Price, ",", "", -1)
			price, err = strconv.ParseFloat(p, 64)
			if err != nil {
				return err
			}
			a := strings.Replace(t.Data.SellQuote[i].Size, ",", "", -1)
			amount, err = strconv.ParseFloat(a, 64)
			if err != nil {
				return err
			}
			if b.orderbookFilter(price, amount) {
				continue
			}
			newOB.Asks = append(newOB.Asks, orderbook.Tranche{
				Price:  price,
				Amount: amount,
			})
		}
		for j := range t.Data.BuyQuote {
			p := strings.Replace(t.Data.BuyQuote[j].Price, ",", "", -1)
			price, err = strconv.ParseFloat(p, 64)
			if err != nil {
				return err
			}
			a := strings.Replace(t.Data.BuyQuote[j].Size, ",", "", -1)
			amount, err = strconv.ParseFloat(a, 64)
			if err != nil {
				return err
			}
			if b.orderbookFilter(price, amount) {
				continue
			}
			newOB.Bids = append(newOB.Bids, orderbook.Tranche{
				Price:  price,
				Amount: amount,
			})
		}
		p, err := currency.NewPairFromString(t.Topic[strings.Index(t.Topic, ":")+1 : strings.Index(t.Topic, currency.UnderscoreDelimiter)])
		if err != nil {
			return err
		}
		var a asset.Item
		a, err = b.GetPairAssetType(p)
		if err != nil {
			return err
		}
		newOB.Pair = p
		newOB.Asset = a
		newOB.Exchange = b.Name
		newOB.Asks.Reverse() // Reverse asks for correct alignment
		newOB.VerifyOrderbook = b.CanVerifyOrderbook
		newOB.LastUpdated = time.Now() // NOTE: Temp to fix test.
		err = b.Websocket.Orderbook.LoadSnapshot(&newOB)
		if err != nil {
			return err
		}
	default:
		return errors.New(b.Name + stream.UnhandledMessage + string(respRaw))
	}

	return nil
}

// orderbookFilter is needed on book levels from this exchange as their data
// is incorrect
func (b *BTSE) orderbookFilter(price, amount float64) bool {
	// Amount filtering occurs when the amount exceeds the decimal returned.
	// e.g. {"price":"1.37","size":"0.00"} currency: SFI-ETH
	// Opted to not round up to 0.01 as this might skew calculations
	// more than removing from the books completely.

	// Price filtering occurs when we are deep in the bid book and there are
	// prices that are less than 4 decimal places
	// e.g. {"price":"0.0000","size":"14219"} currency: TRX-PAX
	// We cannot load a zero price and this will ruin calculations
	return price == 0 || amount == 0
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *BTSE) GenerateDefaultSubscriptions() ([]subscription.Subscription, error) {
	var channels = []string{"orderBookL2Api:%s_0", "tradeHistory:%s"}
	pairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions []subscription.Subscription
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		subscriptions = append(subscriptions, subscription.Subscription{
			Channel: "notificationApi",
		})
	}
	for i := range channels {
		for j := range pairs {
			subscriptions = append(subscriptions, subscription.Subscription{
				Channel: fmt.Sprintf(channels[i], pairs[j]),
				Pair:    pairs[j],
				Asset:   asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (b *BTSE) Subscribe(channelsToSubscribe []subscription.Subscription) error {
	var sub wsSub
	sub.Operation = "subscribe"
	for i := range channelsToSubscribe {
		sub.Arguments = append(sub.Arguments, channelsToSubscribe[i].Channel)
	}
	err := b.Websocket.Conn.SendJSONMessage(sub)
	if err != nil {
		return err
	}
	b.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *BTSE) Unsubscribe(channelsToUnsubscribe []subscription.Subscription) error {
	var unSub wsSub
	unSub.Operation = "unsubscribe"
	for i := range channelsToUnsubscribe {
		unSub.Arguments = append(unSub.Arguments,
			channelsToUnsubscribe[i].Channel)
	}
	err := b.Websocket.Conn.SendJSONMessage(unSub)
	if err != nil {
		return err
	}
	b.Websocket.RemoveSubscriptions(channelsToUnsubscribe...)
	return nil
}
