package btse

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	btseWebsocket = "wss://ws.btse.com/api/ws-feed"
)

// WebsocketSubscriber subscribes to websocket channels with respect to enabled
// currencies
func (b *BTSE) WebsocketSubscriber() error {
	currencies := b.GetEnabledPairs(assets.AssetTypeSpot).Strings()
	subscribe := websocketSubscribe{
		Type: "subscribe",
		Channels: []websocketChannel{
			{
				Name:       "snapshot",
				ProductIDs: currencies,
			},
			{
				Name:       "ticker",
				ProductIDs: currencies,
			},
		},
	}

	data, err := common.JSONEncode(subscribe)
	if err != nil {
		return err
	}

	return b.WebsocketConn.WriteMessage(websocket.TextMessage, data)
}

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

	err = b.WebsocketSubscriber()
	if err != nil {
		return err
	}

	go b.WsHandleData()

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
		err := b.WebsocketConn.Close()
		if err != nil {
			b.Websocket.DataHandler <- fmt.Errorf("%s - Unable to to close Websocket connection. Error: %s",
				b.Name, err)
		}
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

				b.Websocket.DataHandler <- exchange.TickerData{
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

	b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Pair:     p,
		Asset:    "SPOT",
		Exchange: b.GetName(),
	}

	return nil
}
