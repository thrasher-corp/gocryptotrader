package btse

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/exchanges/asset"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	btseWebsocket = "wss://ws.btse.com/api/ws-feed"
)

// WsConnect connects the websocket client
func (b *BTSE) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer

	if b.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(b.Websocket.GetProxyAddress())
		if err != nil {
			return fmt.Errorf("%s websocket error - proxy address %s",
				b.Name, err)
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	b.WebsocketConn, _, err = dialer.Dial(b.Websocket.GetWebsocketURL(),
		http.Header{})
	if err != nil {
		return fmt.Errorf("%s websocket error - unable to connect %s",
			b.Name, err)
	}

	go b.WsHandleData()
	b.GenerateDefaultSubscriptions()

	return nil
}

// WsReadData reads data from the websocket connection
func (b *BTSE) WsReadData() (exchange.WebsocketResponse, error) {
	_, resp, err := b.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	b.Websocket.TrafficAlert <- struct{}{}
	return exchange.WebsocketResponse{Raw: resp}, nil
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
			resp, err := b.WsReadData()
			if err != nil {
				b.Websocket.DataHandler <- err
				return
			}

			type MsgType struct {
				Type      string `json:"type"`
				ProductID string `json:"product_id"`
			}

			if strings.Contains(string(resp.Raw), "Welcome to BTSE") {
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

				b.Websocket.DataHandler <- exchange.TickerData{
					Timestamp:  time.Now(),
					Pair:       currency.NewPairDelimiter(t.ProductID, "-"),
					AssetType:  asset.Spot,
					Exchange:   b.GetName(),
					ClosePrice: price,
					Quantity:   t.LastSize,
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
	base.AssetType = asset.Spot
	base.Pair = p
	base.LastUpdated = time.Now()
	base.ExchangeName = b.Name

	err := base.Process()
	if err != nil {
		return err
	}

	b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Pair:     p,
		Asset:    asset.Spot,
		Exchange: b.GetName(),
	}

	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (b *BTSE) GenerateDefaultSubscriptions() {
	var channels = []string{"snapshot", "ticker"}
	enabledCurrencies := b.GetEnabledPairs(asset.Spot)
	var subscriptions []exchange.WebsocketChannelSubscription
	for i := range channels {
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, exchange.WebsocketChannelSubscription{
				Channel:  channels[i],
				Currency: enabledCurrencies[j],
			})
		}
	}
	b.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (b *BTSE) Subscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	subscribe := websocketSubscribe{
		Type: "subscribe",
		Channels: []websocketChannel{
			{
				Name:       channelToSubscribe.Channel,
				ProductIDs: []string{channelToSubscribe.Currency.String()},
			},
		},
	}
	return b.wsSend(subscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (b *BTSE) Unsubscribe(channelToSubscribe exchange.WebsocketChannelSubscription) error {
	subscribe := websocketSubscribe{
		Type: "unsubscribe",
		Channels: []websocketChannel{
			{
				Name:       channelToSubscribe.Channel,
				ProductIDs: []string{channelToSubscribe.Currency.String()},
			},
		},
	}
	return b.wsSend(subscribe)
}

// WsSend sends data to the websocket server
func (b *BTSE) wsSend(data interface{}) error {
	b.wsRequestMtx.Lock()
	defer b.wsRequestMtx.Unlock()
	if b.Verbose {
		log.Debugf("%v sending message to websocket %v", b.Name, data)
	}
	json, err := common.JSONEncode(data)
	if err != nil {
		return err
	}
	return b.WebsocketConn.WriteMessage(websocket.TextMessage, json)
}
