package coinbasepro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	coinbaseproWebsocketURL = "wss://ws-feed.pro.coinbase.com"
)

// WsConnect initiates a websocket connection
func (c *CoinbasePro) WsConnect() error {
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
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
	case "match", "last_match":
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
		Bids: make(orderbook.Tranches, len(snapshot.Bids)),
		Asks: make(orderbook.Tranches, len(snapshot.Asks)),
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
		base.Bids[i] = orderbook.Tranche{Price: price, Amount: amount}
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
		base.Asks[i] = orderbook.Tranche{Price: price, Amount: amount}
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

	asks := make(orderbook.Tranches, 0, len(update.Changes))
	bids := make(orderbook.Tranches, 0, len(update.Changes))

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
			bids = append(bids, orderbook.Tranche{Price: price, Amount: volume})
		} else {
			asks = append(asks, orderbook.Tranche{Price: price, Amount: volume})
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

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (c *CoinbasePro) generateSubscriptions() (subscription.List, error) {
	pairs, err := c.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	pairFmt, err := c.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	pairs = pairs.Format(pairFmt)
	authed := c.IsWebsocketAuthenticationSupported()
	subs := make(subscription.List, 0, len(c.Features.Subscriptions))
	for _, baseSub := range c.Features.Subscriptions {
		if !authed && baseSub.Authenticated {
			continue
		}

		s := baseSub.Clone()
		s.Asset = asset.Spot
		s.Pairs = pairs
		subs = append(subs, s)
	}
	return subs, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (c *CoinbasePro) Subscribe(subs subscription.List) error {
	r := &WebsocketSubscribe{
		Type:     "subscribe",
		Channels: make([]any, 0, len(subs)),
	}
	// See if we have a consistent Pair list for all the subs that we can use globally
	// If all the subs have the same pairs then we can use the top level ProductIDs field
	// Otherwise each and every sub needs to have it's own list
	for i, s := range subs {
		if i == 0 {
			r.ProductIDs = s.Pairs.Strings()
		} else if !subs[0].Pairs.Equal(s.Pairs) {
			r.ProductIDs = nil
			break
		}
	}
	for _, s := range subs {
		if s.Authenticated && r.Key == "" && c.IsWebsocketAuthenticationSupported() {
			if err := c.authWsSubscibeReq(r); err != nil {
				return err
			}
		}
		if len(r.ProductIDs) == 0 {
			r.Channels = append(r.Channels, WsChannel{
				Name:       s.Channel,
				ProductIDs: s.Pairs.Strings(),
			})
		} else {
			// Coinbase does not support using [WsChannel{Name:"x"}] unless each ProductIDs field is populated
			// Therefore we have to use Channels as an array of strings
			r.Channels = append(r.Channels, s.Channel)
		}
	}
	err := c.Websocket.Conn.SendJSONMessage(context.TODO(), request.Unset, r)
	if err == nil {
		err = c.Websocket.AddSuccessfulSubscriptions(c.Websocket.Conn, subs...)
	}
	return err
}

func (c *CoinbasePro) authWsSubscibeReq(r *WebsocketSubscribe) error {
	creds, err := c.GetCredentials(context.TODO())
	if err != nil {
		return err
	}
	r.Timestamp = strconv.FormatInt(time.Now().Unix(), 10)
	message := r.Timestamp + http.MethodGet + "/users/self/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(message), []byte(creds.Secret))
	if err != nil {
		return err
	}
	r.Signature = crypto.Base64Encode(hmac)
	r.Key = creds.Key
	r.Passphrase = creds.ClientID
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (c *CoinbasePro) Unsubscribe(subs subscription.List) error {
	r := &WebsocketSubscribe{
		Type:     "unsubscribe",
		Channels: make([]any, 0, len(subs)),
	}
	for _, s := range subs {
		r.Channels = append(r.Channels, WsChannel{
			Name:       s.Channel,
			ProductIDs: s.Pairs.Strings(),
		})
	}
	err := c.Websocket.Conn.SendJSONMessage(context.TODO(), request.Unset, r)
	if err == nil {
		err = c.Websocket.RemoveSubscriptions(c.Websocket.Conn, subs...)
	}
	return err
}
