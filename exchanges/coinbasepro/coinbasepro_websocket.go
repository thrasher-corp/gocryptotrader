package coinbasepro

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

const (
	coinbaseproWebsocketURL = "wss://ws-feed.pro.coinbase.com"
)

// WebsocketSubscriber subscribes to websocket channels with respect to enabled
// currencies
func (c *CoinbasePro) WebsocketSubscriber() error {
	currencies := []string{}

	for _, x := range c.GetEnabledPairs(assets.AssetTypeSpot) {
		currency := x.Base.String() + c.CurrencyPairs.RequestFormat.Delimiter +
			x.Quote.String()
		currencies = append(currencies, currency)
	}

	var channels = []WsChannels{
		{
			Name:       "heartbeat",
			ProductIDs: currencies,
		},
		{
			Name:       "ticker",
			ProductIDs: currencies,
		},
		{
			Name:       "level2",
			ProductIDs: currencies,
		},
	}

	subscribe := WebsocketSubscribe{Type: "subscribe", Channels: channels}

	data, err := common.JSONEncode(subscribe)
	if err != nil {
		return err
	}

	return c.WebsocketConn.WriteMessage(websocket.TextMessage, data)
}

// WsConnect initiates a websocket connection
func (c *CoinbasePro) WsConnect() error {
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer

	if c.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(c.Websocket.GetProxyAddress())
		if err != nil {
			return fmt.Errorf("coinbasepro_websocket.go error - proxy address %s",
				err)
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	c.WebsocketConn, _, err = dialer.Dial(c.Websocket.GetWebsocketURL(),
		http.Header{})
	if err != nil {
		return fmt.Errorf("coinbasepro_websocket.go error - unable to connect to websocket %s",
			err)
	}

	err = c.WebsocketSubscriber()
	if err != nil {
		return err
	}

	go c.WsHandleData()

	return nil
}

// WsReadData reads data from the websocket connection
func (c *CoinbasePro) WsReadData() (exchange.WebsocketResponse, error) {
	_, resp, err := c.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	c.Websocket.TrafficAlert <- struct{}{}
	return exchange.WebsocketResponse{Raw: resp}, nil
}

// WsHandleData handles read data from websocket connection
func (c *CoinbasePro) WsHandleData() {
	c.Websocket.Wg.Add(1)

	defer func() {
		err := c.WebsocketConn.Close()
		if err != nil {
			c.Websocket.DataHandler <- fmt.Errorf("coinbasepro_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		c.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-c.Websocket.ShutdownC:
			return

		default:
			resp, err := c.WsReadData()
			if err != nil {
				c.Websocket.DataHandler <- err
				return
			}

			type MsgType struct {
				Type      string `json:"type"`
				Sequence  int64  `json:"sequence"`
				ProductID string `json:"product_id"`
			}

			msgType := MsgType{}
			err = common.JSONDecode(resp.Raw, &msgType)
			if err != nil {
				c.Websocket.DataHandler <- err
				continue
			}

			if msgType.Type == "subscriptions" || msgType.Type == "heartbeat" {
				continue
			}

			switch msgType.Type {
			case "error":
				c.Websocket.DataHandler <- errors.New(string(resp.Raw))

			case "ticker":
				ticker := WebsocketTicker{}
				err := common.JSONDecode(resp.Raw, &ticker)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				c.Websocket.DataHandler <- exchange.TickerData{
					Timestamp:  ticker.Time,
					Pair:       currency.NewPairFromString(ticker.ProductID),
					AssetType:  assets.AssetTypeSpot,
					Exchange:   c.GetName(),
					OpenPrice:  ticker.Open24H,
					HighPrice:  ticker.High24H,
					LowPrice:   ticker.Low24H,
					ClosePrice: ticker.Price,
					Quantity:   ticker.Volume24H,
				}

			case "snapshot":
				snapshot := WebsocketOrderbookSnapshot{}
				err := common.JSONDecode(resp.Raw, &snapshot)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				err = c.ProcessSnapshot(&snapshot)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

			case "l2update":
				update := WebsocketL2Update{}
				err := common.JSONDecode(resp.Raw, &update)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				err = c.ProcessUpdate(update)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
			}
		}
	}
}

// ProcessSnapshot processes the initial orderbook snap shot
func (c *CoinbasePro) ProcessSnapshot(snapshot *WebsocketOrderbookSnapshot) error {
	var base orderbook.Base
	for _, bid := range snapshot.Bids {
		price, err := strconv.ParseFloat(bid[0].(string), 64)
		if err != nil {
			return err
		}

		amount, err := strconv.ParseFloat(bid[1].(string), 64)
		if err != nil {
			return err
		}

		base.Bids = append(base.Bids,
			orderbook.Item{Price: price, Amount: amount})
	}

	for _, ask := range snapshot.Asks {
		price, err := strconv.ParseFloat(ask[0].(string), 64)
		if err != nil {
			return err
		}

		amount, err := strconv.ParseFloat(ask[1].(string), 64)
		if err != nil {
			return err
		}

		base.Asks = append(base.Asks,
			orderbook.Item{Price: price, Amount: amount})
	}

	p := currency.NewPairFromString(snapshot.ProductID)
	base.AssetType = assets.AssetTypeSpot
	base.Pair = p
	base.LastUpdated = time.Now()

	err := c.Websocket.Orderbook.LoadSnapshot(&base, c.GetName(), false)
	if err != nil {
		return err
	}

	c.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Pair:     p,
		Asset:    assets.AssetTypeSpot,
		Exchange: c.GetName(),
	}

	return nil
}

// ProcessUpdate updates the orderbook local cache
func (c *CoinbasePro) ProcessUpdate(update WebsocketL2Update) error {
	var Asks, Bids []orderbook.Item

	for _, data := range update.Changes {
		price, _ := strconv.ParseFloat(data[1].(string), 64)
		volume, _ := strconv.ParseFloat(data[2].(string), 64)

		if data[0].(string) == exchange.BuyOrderSide.ToLower().ToString() {
			Bids = append(Bids, orderbook.Item{Price: price, Amount: volume})
		} else {
			Asks = append(Asks, orderbook.Item{Price: price, Amount: volume})
		}
	}

	if len(Asks) == 0 && len(Bids) == 0 {
		return errors.New("coibasepro_websocket.go error - no data in websocket update")
	}

	p := currency.NewPairFromString(update.ProductID)

	err := c.Websocket.Orderbook.Update(Bids, Asks, p, time.Now(), c.GetName(), assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	c.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Pair:     p,
		Asset:    assets.AssetTypeSpot,
		Exchange: c.GetName(),
	}

	return nil
}
