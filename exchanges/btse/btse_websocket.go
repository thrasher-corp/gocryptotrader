package btse

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
)

const (
	btseWebsocket      = "wss://ws.btse.com/spotWS"
	btseWebsocketTimer = 57 * time.Second
)

// WsConnect connects the websocket client
func (b *BTSE) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := b.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	b.WebsocketConn.SetupPingHandler(wshandler.WebsocketPingHandler{
		MessageType: websocket.PingMessage,
		Delay:       btseWebsocketTimer,
	})

	go b.wsReadData()
	if b.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err = b.WsAuthenticate()
		if err != nil {
			b.Websocket.DataHandler <- err
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	b.GenerateDefaultSubscriptions()
	return nil
}

// WsAuthenticate Send an authentication message to receive auth data
func (b *BTSE) WsAuthenticate() error {
	nonce := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	path := "/spotWS" + nonce
	hmac := crypto.GetHMAC(
		crypto.HashSHA512_384,
		[]byte((path + nonce)),
		[]byte(b.API.Credentials.Secret),
	)
	sign := crypto.HexEncodeToString(hmac)
	req := wsSub{
		Operation: "authKeyExpires",
		Arguments: []string{b.API.Credentials.Key, nonce, sign},
	}
	return b.WebsocketConn.SendJSONMessage(req)
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
	b.Websocket.Wg.Add(1)

	defer func() {
		b.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		default:
			resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				b.Websocket.ReadMessageErrors <- err
				return
			}
			b.Websocket.TrafficAlert <- struct{}{}
			err = b.wsHandleData(resp.Raw)
			if err != nil {
				b.Websocket.DataHandler <- err
			}
		}
	}
}

func (b *BTSE) wsHandleData(respRaw []byte) error {
	type Result map[string]interface{}
	var result Result
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	switch {
	case result["topic"] == "notificationApi":
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
			p := currency.NewPairFromString(notification.Data[i].Symbol)
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
				ID:           notification.Data[i].OrderID,
				Type:         oType,
				Side:         oSide,
				Status:       oStatus,
				AssetType:    a,
				Date:         time.Unix(0, notification.Data[i].Timestamp*int64(time.Millisecond)),
				Pair:         p,
			}
		}

	case strings.Contains(result["topic"].(string), "tradeHistory"):
		var tradeHistory wsTradeHistory
		err = json.Unmarshal(respRaw, &tradeHistory)
		if err != nil {
			return err
		}
		for x := range tradeHistory.Data {
			side := order.Buy
			if tradeHistory.Data[x].Gain == -1 {
				side = order.Sell
			}
			p := currency.NewPairFromString(strings.Replace(tradeHistory.Topic, "tradeHistory:", "", 1))
			var a asset.Item
			a, err = b.GetPairAssetType(p)
			if err != nil {
				return err
			}
			b.Websocket.DataHandler <- wshandler.TradeData{
				Timestamp:    time.Unix(0, tradeHistory.Data[x].TransactionTime*int64(time.Millisecond)),
				CurrencyPair: p,
				AssetType:    a,
				Exchange:     b.Name,
				Price:        tradeHistory.Data[x].Price,
				Amount:       tradeHistory.Data[x].Amount,
				Side:         side,
			}
		}
	case strings.Contains(result["topic"].(string), "orderBookApi"):
		var t wsOrderBook
		err = json.Unmarshal(respRaw, &t)
		if err != nil {
			return err
		}
		var newOB orderbook.Base
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
			newOB.Asks = append(newOB.Asks, orderbook.Item{
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
			newOB.Bids = append(newOB.Bids, orderbook.Item{
				Price:  price,
				Amount: amount,
			})
		}
		p := currency.NewPairFromString(t.Topic[strings.Index(t.Topic, ":")+1 : strings.Index(t.Topic, "_")])
		var a asset.Item
		a, err = b.GetPairAssetType(p)
		if err != nil {
			return err
		}
		newOB.Pair = p
		newOB.AssetType = a
		newOB.ExchangeName = b.Name
		err = b.Websocket.Orderbook.LoadSnapshot(&newOB)
		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: newOB.Pair,
			Asset:    a,
			Exchange: b.Name}
	default:
		b.Websocket.DataHandler <- wshandler.UnhandledMessageWarning{Message: b.Name + wshandler.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *BTSE) GenerateDefaultSubscriptions() {
	var channels = []string{"orderBookApi:%s_0", "tradeHistory:%s"}
	pairs := b.GetEnabledPairs(asset.Spot)
	var subscriptions []wshandler.WebsocketChannelSubscription
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
			Channel: "notificationApi",
		})
	}
	for i := range channels {
		for j := range pairs {
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  fmt.Sprintf(channels[i], pairs[j]),
				Currency: pairs[j],
			})
		}
	}
	b.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (b *BTSE) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var sub wsSub
	sub.Operation = "subscribe"
	sub.Arguments = []string{channelToSubscribe.Channel}

	return b.WebsocketConn.SendJSONMessage(sub)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *BTSE) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var unSub wsSub
	unSub.Operation = "unsubscribe"
	unSub.Arguments = []string{channelToSubscribe.Channel}
	return b.WebsocketConn.SendJSONMessage(unSub)
}
