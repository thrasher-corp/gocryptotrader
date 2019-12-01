package coinbasepro

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
)

const (
	coinbaseproWebsocketURL = "wss://ws-feed.pro.coinbase.com"
)

// WsConnect initiates a websocket connection
func (c *CoinbasePro) WsConnect() error {
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return errors.New(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := c.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	c.GenerateDefaultSubscriptions()
	go c.WsHandleData()

	return nil
}

// WsHandleData handles read data from websocket connection
func (c *CoinbasePro) WsHandleData() {
	c.Websocket.Wg.Add(1)

	defer func() {
		c.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-c.Websocket.ShutdownC:
			return
		default:
			resp, err := c.WebsocketConn.ReadMessage()
			if err != nil {
				c.Websocket.ReadMessageErrors <- err
				return
			}
			c.Websocket.TrafficAlert <- struct{}{}

			type MsgType struct {
				Type      string `json:"type"`
				Sequence  int64  `json:"sequence"`
				ProductID string `json:"product_id"`
			}

			msgType := MsgType{}
			err = json.Unmarshal(resp.Raw, &msgType)
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
				err := json.Unmarshal(resp.Raw, &ticker)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				c.Websocket.DataHandler <- wshandler.TickerData{
					Timestamp: ticker.Time,
					Pair:      ticker.ProductID,
					AssetType: asset.Spot,
					Exchange:  c.Name,
					Open:      ticker.Open24H,
					High:      ticker.High24H,
					Low:       ticker.Low24H,
					Last:      ticker.Price,
					Volume:    ticker.Volume24H,
					Bid:       ticker.BestBid,
					Ask:       ticker.BestAsk,
				}

			case "snapshot":
				snapshot := WebsocketOrderbookSnapshot{}
				err := json.Unmarshal(resp.Raw, &snapshot)
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
				err := json.Unmarshal(resp.Raw, &update)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}

				err = c.ProcessUpdate(update)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
			case "received":
				// We currently use l2update to calculate orderbook changes
				received := WebsocketReceived{}
				err := json.Unmarshal(resp.Raw, &received)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- received
			case "open":
				// We currently use l2update to calculate orderbook changes
				open := WebsocketOpen{}
				err := json.Unmarshal(resp.Raw, &open)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- open
			case "done":
				// We currently use l2update to calculate orderbook changes
				done := WebsocketDone{}
				err := json.Unmarshal(resp.Raw, &done)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- done
			case "change":
				// We currently use l2update to calculate orderbook changes
				change := WebsocketChange{}
				err := json.Unmarshal(resp.Raw, &change)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- change
			case "activate":
				// We currently use l2update to calculate orderbook changes
				activate := WebsocketActivate{}
				err := json.Unmarshal(resp.Raw, &activate)
				if err != nil {
					c.Websocket.DataHandler <- err
					continue
				}
				c.Websocket.DataHandler <- activate
			}
		}
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

	pair := currency.NewPairFromString(snapshot.ProductID)
	base.AssetType = asset.Spot
	base.Pair = pair
	base.ExchangeName = c.Name

	err := c.Websocket.Orderbook.LoadSnapshot(&base)
	if err != nil {
		return err
	}

	c.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Pair:     pair,
		Asset:    asset.Spot,
		Exchange: c.Name,
	}

	return nil
}

// ProcessUpdate updates the orderbook local cache
func (c *CoinbasePro) ProcessUpdate(update WebsocketL2Update) error {
	var asks, bids []orderbook.Item

	for i := range update.Changes {
		price, _ := strconv.ParseFloat(update.Changes[i][1].(string), 64)
		volume, _ := strconv.ParseFloat(update.Changes[i][2].(string), 64)

		if update.Changes[i][0].(string) == order.Buy.Lower() {
			bids = append(bids, orderbook.Item{Price: price, Amount: volume})
		} else {
			asks = append(asks, orderbook.Item{Price: price, Amount: volume})
		}
	}

	if len(asks) == 0 && len(bids) == 0 {
		return errors.New("coinbasepro_websocket.go error - no data in websocket update")
	}

	p := currency.NewPairFromString(update.ProductID)
	timestamp, err := time.Parse(time.RFC3339, update.Time)
	if err != nil {
		return err
	}
	err = c.Websocket.Orderbook.Update(&wsorderbook.WebsocketOrderbookUpdate{
		Bids:       bids,
		Asks:       asks,
		Pair:       p,
		UpdateTime: timestamp,
		Asset:      asset.Spot,
	})
	if err != nil {
		return err
	}

	c.Websocket.DataHandler <- wshandler.WebsocketOrderbookUpdate{
		Pair:     p,
		Asset:    asset.Spot,
		Exchange: c.Name,
	}

	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (c *CoinbasePro) GenerateDefaultSubscriptions() {
	var channels = []string{"heartbeat", "level2", "ticker", "user"}
	enabledCurrencies := c.GetEnabledPairs(asset.Spot)
	var subscriptions []wshandler.WebsocketChannelSubscription
	for i := range channels {
		if (channels[i] == "user" || channels[i] == "full") && !c.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
			continue
		}
		for j := range enabledCurrencies {
			subscriptions = append(subscriptions, wshandler.WebsocketChannelSubscription{
				Channel: channels[i],
				Currency: c.FormatExchangeCurrency(enabledCurrencies[j],
					asset.Spot),
			})
		}
	}
	c.Websocket.SubscribeToChannels(subscriptions)
}

// Subscribe sends a websocket message to receive data from the channel
func (c *CoinbasePro) Subscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	subscribe := WebsocketSubscribe{
		Type: "subscribe",
		Channels: []WsChannels{
			{
				Name: channelToSubscribe.Channel,
				ProductIDs: []string{
					c.FormatExchangeCurrency(channelToSubscribe.Currency,
						asset.Spot).String(),
				},
			},
		},
	}
	if channelToSubscribe.Channel == "user" || channelToSubscribe.Channel == "full" {
		n := fmt.Sprintf("%v", time.Now().Unix())
		message := n + "GET" + "/users/self/verify"
		hmac := crypto.GetHMAC(crypto.HashSHA256, []byte(message),
			[]byte(c.API.Credentials.Secret))
		subscribe.Signature = crypto.Base64Encode(hmac)
		subscribe.Key = c.API.Credentials.Key
		subscribe.Passphrase = c.API.Credentials.ClientID
		subscribe.Timestamp = n
	}
	return c.WebsocketConn.SendMessage(subscribe)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (c *CoinbasePro) Unsubscribe(channelToSubscribe wshandler.WebsocketChannelSubscription) error {
	subscribe := WebsocketSubscribe{
		Type: "unsubscribe",
		Channels: []WsChannels{
			{
				Name: channelToSubscribe.Channel,
				ProductIDs: []string{
					c.FormatExchangeCurrency(channelToSubscribe.Currency,
						asset.Spot).String(),
				},
			},
		},
	}
	return c.WebsocketConn.SendMessage(subscribe)
}
