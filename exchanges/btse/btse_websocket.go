package btse

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	btseWebsocket = "wss://ws.btse.com/spotWS"
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
	go b.Pinger()
	go b.WsHandleData()
	b.GenerateDefaultSubscriptions()

	return nil
}

// WsHandleData handles read data from websocket connection
func (b *BTSE) WsHandleData() {
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
				b.Websocket.DataHandler <- err
				return
			}
			b.Websocket.TrafficAlert <- struct{}{}

			type Result map[string]interface{}

			if strings.Contains(string(resp.Raw), "Connected. Welcome to BTSE!") {
				if b.Verbose {
					log.Debugf("%s websocket client successfully connected to %s",
						b.Name, b.Websocket.GetWebsocketURL())
				}
				continue
			}

			result := Result{}
			err = common.JSONDecode(resp.Raw, &result)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}
			switch {
			case strings.Contains(result["topic"].(string), "tradeHistory"):
				var tradeHistory wsTradeHistory
				err = common.JSONDecode(resp.Raw, &tradeHistory)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}
				for x := range tradeHistory.Data {
					side := "buy"
					if tradeHistory.Data[x].Gain == -1 {
						side = "sell"
					}
					b.Websocket.DataHandler <- wshandler.TradeData{
						Timestamp:    time.Unix(tradeHistory.Data[x].TrasnsactionTime, 0),
						CurrencyPair: currency.NewPairFromString(strings.Replace(tradeHistory.Topic, "tradeHistory", "", 1)),
						AssetType:    orderbook.Spot,
						Exchange:     b.Name,
						Price:        tradeHistory.Data[x].Price,
						Amount:       tradeHistory.Data[x].Amount,
						Side:         side,
					}
				}
			case strings.Contains(result["topic"].(string), "orderBookApi"):
				var t wsOrderBook
				err = common.JSONDecode(resp.Raw, &t)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}
				var price, amount float64
				var asks, bids []orderbook.Item
				for i := range t.Data.BuyQuote {
					p := strings.Replace(t.Data.BuyQuote[i].Price, ",", "", -1)
					price, err = strconv.ParseFloat(p, 64)
					if err != nil {
						b.Websocket.DataHandler <- err
					}
					a := strings.Replace(t.Data.BuyQuote[i].Size, ",", "", -1)
					amount, err = strconv.ParseFloat(a, 64)
					if err != nil {
						b.Websocket.DataHandler <- err
					}
					asks = append(asks, orderbook.Item{Price: price, Amount: amount})
				}
				for j := range t.Data.SellQuote {
					p := strings.Replace(t.Data.SellQuote[j].Price, ",", "", -1)
					price, err = strconv.ParseFloat(p, 64)
					if err != nil {
						b.Websocket.DataHandler <- err
					}
					a := strings.Replace(t.Data.SellQuote[j].Size, ",", "", -1)
					amount, err = strconv.ParseFloat(a, 64)
					if err != nil {
						b.Websocket.DataHandler <- err
					}
					bids = append(bids, orderbook.Item{Price: price, Amount: amount})
				}
				var newOB orderbook.Base
				newOB.Asks = asks
				newOB.Bids = bids
				newOB.AssetType = orderbook.Spot
				newOB.Pair = currency.NewPairFromString(t.Topic[strings.Index(t.Topic, ":")+1 : strings.Index(t.Topic, "_")])
				newOB.ExchangeName = b.Name
				err = b.Websocket.Orderbook.LoadSnapshot(&newOB, true)
				if err != nil {
					b.Websocket.DataHandler <- err
				}
				b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{Pair: newOB.Pair,
					Asset:    orderbook.Spot,
					Exchange: b.Name}
			}
		}
	}
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *BTSE) GenerateDefaultSubscriptions() {
	var channels = []string{"orderBookApi:%s_0", "tradeHistory:%s"}
	var subscriptions []wshandler.WebsocketChannelSubscription
	for i := range channels {
		for j := range b.EnabledPairs {
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  fmt.Sprintf(channels[i], b.EnabledPairs[j]),
				Currency: b.EnabledPairs[j],
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
	return b.WebsocketConn.SendMessage(sub)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *BTSE) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	var unSub wsSub
	unSub.Operation = "unsubscribe"
	unSub.Arguments = []string{channelToSubscribe.Channel}
	return b.WebsocketConn.SendMessage(unSub)
}

// Pinger pings
func (b *BTSE) Pinger() {
	b.Websocket.Wg.Add(1)

	defer b.Websocket.Wg.Done()
	count := 57 * time.Second
	timer := time.NewTimer(count)

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		case <-timer.C:
			b.WebsocketConn.Connection.WriteMessage(websocket.TextMessage, []byte("ping"))
			timer.Reset(count)
		}
	}
}
