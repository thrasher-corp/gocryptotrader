package btse

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/wshandler"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	btseWebsocket = "wss://ws.btse.com/api/ws-feed"
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

			type MsgType struct {
				Type      string `json:"type"`
				ProductID string `json:"product_id"`
			}

			if strings.Contains(string(resp.Raw), "connect success") {
				if b.Verbose {
					log.Debugf("%s websocket client successfully connected to %s",
						b.Name, b.Websocket.GetWebsocketURL())
				}
				continue
			}

			msgType := MsgType{}
			err = common.JSONDecode(resp.Raw, &msgType)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}
			switch msgType.Type {
			case "ticker":
				var t wsTicker
				err = common.JSONDecode(resp.Raw, &t)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}
				p := strings.Replace(t.Price.(string), ",", "", -1)
				price, err := strconv.ParseFloat(p, 64)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				b.Websocket.DataHandler <- wshandler.TickerData{
					Timestamp: time.Now(),
					Pair:      currency.NewPairDelimiter(t.ProductID, "-"),
					AssetType: "SPOT",
					Exchange:  b.GetName(),
					OpenPrice: price,
				}
			case "snapshot":
				snapshot := websocketOrderbookSnapshot{}
				err := common.JSONDecode(resp.Raw, &snapshot)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				err = b.wsProcessSnapshot(&snapshot)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}
			}
		}
	}
}

// ProcessSnapshot processes the initial orderbook snap shot
func (b *BTSE) wsProcessSnapshot(snapshot *websocketOrderbookSnapshot) error {
	var base orderbook.Base
	for _, bid := range snapshot.Bids {
		p := strings.Replace(bid[0].(string), ",", "", -1)
		price, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return err
		}

		a := strings.Replace(bid[1].(string), ",", "", -1)
		amount, err := strconv.ParseFloat(a, 64)
		if err != nil {
			return err
		}

		base.Bids = append(base.Bids,
			orderbook.Item{Price: price, Amount: amount})
	}

	for _, ask := range snapshot.Asks {
		p := strings.Replace(ask[0].(string), ",", "", -1)
		price, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return err
		}

		a := strings.Replace(ask[1].(string), ",", "", -1)
		amount, err := strconv.ParseFloat(a, 64)
		if err != nil {
			return err
		}

		base.Asks = append(base.Asks,
			orderbook.Item{Price: price, Amount: amount})
	}

	p := currency.NewPairDelimiter(snapshot.ProductID, "-")
	base.AssetType = "SPOT"
	base.Pair = p
	base.LastUpdated = time.Now()
	base.ExchangeName = b.Name

	err := base.Process()
	if err != nil {
		return err
	}

	b.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Pair:     p,
		Asset:    "SPOT",
		Exchange: b.GetName(),
	}

	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *BTSE) GenerateDefaultSubscriptions() {
	var channels = []string{"snapshot", "ticker"}
	enabledCurrencies := b.GetEnabledCurrencies()
	var subscriptions []wshandler.WebsocketChannelSubscription
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	b.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (b *BTSE) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	subscribe := websocketSubscribe{
		Type: "subscribe",
		Channels: []websocketChannel{
			{
				Name:       channelToSubscribe.Channel,
				ProductIDs: []string{channelToSubscribe.Currency.String()},
			},
		},
	}
	return b.WebsocketConn.SendMessage(subscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *BTSE) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	subscribe := websocketSubscribe{
		Type: "unsubscribe",
		Channels: []websocketChannel{
			{
				Name:       channelToSubscribe.Channel,
				ProductIDs: []string{channelToSubscribe.Currency.String()},
			},
		},
	}
	return b.WebsocketConn.SendMessage(subscribe)
}
