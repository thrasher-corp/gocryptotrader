package coinbasepro

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	coinbaseproWebsocketURL = "wss://ws-feed.pro.coinbase.com"
)

// WsConnect initiates a websocket connection
func (e *Exchange) WsConnect() error {
	ctx := context.TODO()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}

	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (e *Exchange) wsReadData(ctx context.Context) {
	defer e.Websocket.Wg.Done()

	for {
		resp := e.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := e.wsHandleData(ctx, resp.Raw)
		if err != nil {
			e.Websocket.DataHandler <- err
		}
	}
}

func (e *Exchange) wsHandleData(ctx context.Context, respRaw []byte) error {
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
		e.Websocket.DataHandler <- status
	case "error":
		e.Websocket.DataHandler <- errors.New(string(respRaw))
	case "ticker":
		wsTicker := WebsocketTicker{}
		err := json.Unmarshal(respRaw, &wsTicker)
		if err != nil {
			return err
		}

		e.Websocket.DataHandler <- &ticker.Price{
			LastUpdated:  wsTicker.Time,
			Pair:         wsTicker.ProductID,
			AssetType:    asset.Spot,
			ExchangeName: e.Name,
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

		err = e.ProcessSnapshot(&snapshot)
		if err != nil {
			return err
		}
	case "l2update":
		var update WebsocketL2Update
		err := json.Unmarshal(respRaw, &update)
		if err != nil {
			return err
		}

		err = e.ProcessOrderbookUpdate(&update)
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
			e.Websocket.DataHandler <- order.ClassificationError{
				Exchange: e.Name,
				OrderID:  wsOrder.OrderID,
				Err:      err,
			}
		}
		oSide, err = order.StringToOrderSide(wsOrder.Side)
		if err != nil {
			e.Websocket.DataHandler <- order.ClassificationError{
				Exchange: e.Name,
				OrderID:  wsOrder.OrderID,
				Err:      err,
			}
		}
		oStatus, err = statusToStandardStatus(wsOrder.Type)
		if err != nil {
			e.Websocket.DataHandler <- order.ClassificationError{
				Exchange: e.Name,
				OrderID:  wsOrder.OrderID,
				Err:      err,
			}
		}
		if wsOrder.Reason == "canceled" {
			oStatus = order.Cancelled
		}
		ts := wsOrder.Time
		if wsOrder.Type == "activate" {
			ts = wsOrder.Timestamp.Time()
		}

		creds, err := e.GetCredentials(ctx)
		if err != nil {
			e.Websocket.DataHandler <- order.ClassificationError{
				Exchange: e.Name,
				OrderID:  wsOrder.OrderID,
				Err:      err,
			}
		}

		clientID := ""
		if creds != nil {
			clientID = creds.ClientID
		}

		if wsOrder.UserID != "" {
			var p currency.Pair
			var a asset.Item
			p, a, err = e.GetRequestFormattedPairAndAssetType(wsOrder.ProductID)
			if err != nil {
				return err
			}
			e.Websocket.DataHandler <- &order.Detail{
				HiddenOrder:     wsOrder.Private,
				Price:           wsOrder.Price,
				Amount:          wsOrder.Size,
				TriggerPrice:    wsOrder.StopPrice,
				ExecutedAmount:  wsOrder.Size - wsOrder.RemainingSize,
				RemainingAmount: wsOrder.RemainingSize,
				Fee:             wsOrder.TakerFeeRate,
				Exchange:        e.Name,
				OrderID:         wsOrder.OrderID,
				AccountID:       wsOrder.ProfileID,
				ClientID:        clientID,
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
			e.Websocket.DataHandler <- order.ClassificationError{
				Exchange: e.Name,
				Err:      err,
			}
		}
		var p currency.Pair
		var a asset.Item
		p, a, err = e.GetRequestFormattedPairAndAssetType(wsOrder.ProductID)
		if err != nil {
			return err
		}

		if wsOrder.UserID != "" {
			e.Websocket.DataHandler <- &order.Detail{
				OrderID:   wsOrder.OrderID,
				Pair:      p,
				AssetType: a,
				Trades: []order.TradeHistory{
					{
						Price:     wsOrder.Price,
						Amount:    wsOrder.Size,
						Exchange:  e.Name,
						TID:       strconv.FormatInt(wsOrder.TradeID, 10),
						Side:      oSide,
						Timestamp: wsOrder.Time,
						IsMaker:   wsOrder.TakerUserID == "",
					},
				},
			}
		} else {
			if !e.IsSaveTradeDataEnabled() {
				return nil
			}
			return trade.AddTradesToBuffer(trade.Data{
				Timestamp:    wsOrder.Time,
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        wsOrder.Price,
				Amount:       wsOrder.Size,
				Side:         oSide,
				TID:          strconv.FormatInt(wsOrder.TradeID, 10),
			})
		}
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
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
func (e *Exchange) ProcessSnapshot(snapshot *WebsocketOrderbookSnapshot) error {
	pair, err := currency.NewPairFromString(snapshot.ProductID)
	if err != nil {
		return err
	}

	ob := &orderbook.Book{
		Pair:              pair,
		Bids:              make(orderbook.Levels, len(snapshot.Bids)),
		Asks:              make(orderbook.Levels, len(snapshot.Asks)),
		Asset:             asset.Spot,
		Exchange:          e.Name,
		ValidateOrderbook: e.ValidateOrderbook,
		LastUpdated:       snapshot.Time,
	}

	for i := range snapshot.Bids {
		ob.Bids[i].Price = snapshot.Bids[i][0].Float64()
		ob.Bids[i].Amount = snapshot.Bids[i][1].Float64()
	}
	for i := range snapshot.Asks {
		ob.Asks[i].Price = snapshot.Asks[i][0].Float64()
		ob.Asks[i].Amount = snapshot.Asks[i][1].Float64()
	}
	return e.Websocket.Orderbook.LoadSnapshot(ob)
}

// ProcessOrderbookUpdate updates the orderbook local cache
func (e *Exchange) ProcessOrderbookUpdate(update *WebsocketL2Update) error {
	if len(update.Changes) == 0 {
		return errors.New("no data in websocket update")
	}

	p, err := currency.NewPairFromString(update.ProductID)
	if err != nil {
		return err
	}

	asks := make(orderbook.Levels, 0, len(update.Changes))
	bids := make(orderbook.Levels, 0, len(update.Changes))

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
			bids = append(bids, orderbook.Level{Price: price, Amount: volume})
		} else {
			asks = append(asks, orderbook.Level{Price: price, Amount: volume})
		}
	}

	return e.Websocket.Orderbook.Update(&orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       p,
		UpdateTime: update.Time,
		Asset:      asset.Spot,
	})
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	pairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	pairFmt, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	pairs = pairs.Format(pairFmt)
	authed := e.IsWebsocketAuthenticationSupported()
	subs := make(subscription.List, 0, len(e.Features.Subscriptions))
	for _, baseSub := range e.Features.Subscriptions {
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
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
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
		if s.Authenticated && r.Key == "" && e.IsWebsocketAuthenticationSupported() {
			if err := e.authWsSubscibeReq(ctx, r); err != nil {
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
	err := e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, r)
	if err == nil {
		err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, subs...)
	}
	return err
}

func (e *Exchange) authWsSubscibeReq(ctx context.Context, r *WebsocketSubscribe) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	r.Timestamp = strconv.FormatInt(time.Now().Unix(), 10)
	message := r.Timestamp + http.MethodGet + "/users/self/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(message), []byte(creds.Secret))
	if err != nil {
		return err
	}
	r.Signature = base64.StdEncoding.EncodeToString(hmac)
	r.Key = creds.Key
	r.Passphrase = creds.ClientID
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
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
	err := e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, r)
	if err == nil {
		err = e.Websocket.RemoveSubscriptions(e.Websocket.Conn, subs...)
	}
	return err
}
