package btcmarkets

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
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
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const btcMarketsWSURL = "wss://socket.btcmarkets.net/v2"

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: true, Channel: subscription.HeartbeatChannel},
}

var subscriptionNames = map[string]string{
	subscription.OrderbookChannel: wsOrderbookUpdate,
	subscription.TickerChannel:    tick,
	subscription.AllTradesChannel: tradeEndPoint,
	subscription.MyOrdersChannel:  orderChange,
	subscription.MyAccountChannel: fundChange,
	subscription.HeartbeatChannel: heartbeat,
}

// WsConnect connects to a websocket feed
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
	if e.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", e.Name)
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
		if err := e.wsHandleData(ctx, resp.Raw); err != nil {
			if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
				log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
			}
		}
	}
}

// UnmarshalJSON implements the unmarshaler interface.
func (w *WebsocketOrderbook) UnmarshalJSON(data []byte) error {
	var resp [][3]types.Number
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	*w = WebsocketOrderbook(make(orderbook.Levels, len(resp)))
	for x := range resp {
		(*w)[x].Price = resp[x][0].Float64()
		(*w)[x].Amount = resp[x][1].Float64()
		(*w)[x].OrderCount = resp[x][2].Int64()
	}
	return nil
}

func (e *Exchange) wsHandleData(ctx context.Context, respRaw []byte) error {
	var wsResponse WsMessageType
	err := json.Unmarshal(respRaw, &wsResponse)
	if err != nil {
		return err
	}
	switch wsResponse.MessageType {
	case heartbeat:
		if e.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket heartbeat received %s", e.Name, respRaw)
		}
	case wsOrderbookUpdate:
		var ob WsOrderbook
		err := json.Unmarshal(respRaw, &ob)
		if err != nil {
			return err
		}

		if ob.Snapshot {
			err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
				Pair:              ob.Currency,
				Bids:              orderbook.Levels(ob.Bids),
				Asks:              orderbook.Levels(ob.Asks),
				LastUpdated:       ob.Timestamp,
				LastUpdateID:      ob.SnapshotID,
				Asset:             asset.Spot,
				Exchange:          e.Name,
				ValidateOrderbook: e.ValidateOrderbook,
			})
		} else {
			err = e.Websocket.Orderbook.Update(&orderbook.Update{
				UpdateTime:                 ob.Timestamp,
				UpdateID:                   ob.SnapshotID,
				Asset:                      asset.Spot,
				Bids:                       orderbook.Levels(ob.Bids),
				Asks:                       orderbook.Levels(ob.Asks),
				Pair:                       ob.Currency,
				ExpectedChecksum:           ob.Checksum,
				GenerateChecksum:           orderbookChecksum,
				SkipOutOfOrderLastUpdateID: true,
			})
		}
		if err != nil {
			if errors.Is(err, orderbook.ErrOrderbookInvalid) {
				err2 := e.ReSubscribeSpecificOrderbook(ob.Currency)
				if err2 != nil {
					return err2
				}
			}
			return err
		}
		return nil
	case tradeEndPoint:
		tradeFeed := e.IsTradeFeedEnabled()
		saveTradeData := e.IsSaveTradeDataEnabled()
		if !saveTradeData && !tradeFeed {
			return nil
		}

		var t WsTrade
		err := json.Unmarshal(respRaw, &t)
		if err != nil {
			return err
		}

		side := order.Buy
		switch {
		case t.Side.IsLong():
			// Nothing to do
		case t.Side.IsShort():
			side = order.Sell
		default:
			return fmt.Errorf("%w: %q", order.ErrSideIsInvalid, t.Side)
		}

		td := trade.Data{
			Timestamp:    t.Timestamp,
			CurrencyPair: t.MarketID,
			AssetType:    asset.Spot,
			Exchange:     e.Name,
			Price:        t.Price,
			Amount:       t.Volume,
			Side:         side,
			TID:          strconv.FormatInt(t.TradeID, 10),
		}

		if tradeFeed {
			if err := e.Websocket.DataHandler.Send(ctx, td); err != nil {
				return err
			}
		}
		if saveTradeData {
			return trade.AddTradesToBuffer(td)
		}
	case tick:
		var tick WsTick
		err := json.Unmarshal(respRaw, &tick)
		if err != nil {
			return err
		}

		return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
			ExchangeName: e.Name,
			Volume:       tick.Volume,
			High:         tick.High24,
			Low:          tick.Low24h,
			Bid:          tick.Bid,
			Ask:          tick.Ask,
			Last:         tick.Last,
			LastUpdated:  tick.Timestamp,
			AssetType:    asset.Spot,
			Pair:         tick.MarketID,
		})
	case fundChange:
		var transferData WsFundTransfer
		err := json.Unmarshal(respRaw, &transferData)
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, transferData)
	case orderChange:
		var orderData WsOrderChange
		err := json.Unmarshal(respRaw, &orderData)
		if err != nil {
			return err
		}
		originalAmount := orderData.OpenVolume
		var price float64
		var trades []order.TradeHistory
		orderID := strconv.FormatInt(orderData.OrderID, 10)
		for x := range orderData.Trades {
			var isMaker bool
			if orderData.Trades[x].LiquidityType == "Maker" {
				isMaker = true
			}
			trades = append(trades, order.TradeHistory{
				Price:    orderData.Trades[x].Price,
				Amount:   orderData.Trades[x].Volume,
				Fee:      orderData.Trades[x].Fee,
				Exchange: e.Name,
				TID:      strconv.FormatInt(orderData.Trades[x].TradeID, 10),
				IsMaker:  isMaker,
			})
			price = orderData.Trades[x].Price
			originalAmount += orderData.Trades[x].Volume
		}
		oType, err := order.StringToOrderType(orderData.OrderType)
		if err != nil {
			return err
		}
		oSide, err := order.StringToOrderSide(orderData.Side)
		if err != nil {
			return err
		}
		oStatus, err := order.StringToOrderStatus(orderData.Status)
		if err != nil {
			return err
		}

		clientID := ""
		if creds, err := e.GetCredentials(ctx); err != nil {
			return err
		} else if creds != nil {
			clientID = creds.ClientID
		}

		return e.Websocket.DataHandler.Send(ctx, &order.Detail{
			Price:           price,
			Amount:          originalAmount,
			RemainingAmount: orderData.OpenVolume,
			Exchange:        e.Name,
			OrderID:         orderID,
			ClientID:        clientID,
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       asset.Spot,
			Date:            orderData.Timestamp,
			Trades:          trades,
			Pair:            orderData.MarketID,
		})
	case "error":
		var wsErr WsError
		err := json.Unmarshal(respRaw, &wsErr)
		if err != nil {
			return err
		}
		return fmt.Errorf("%v websocket error. Code: %v Message: %v", e.Name, wsErr.Code, wsErr.Message)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
	}
	return nil
}

func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{"channelName": channelName}).Parse(subTplText)
}

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	baseReq := &WsSubscribe{
		MessageType: subscribe,
	}

	var errs error
	if authed := subs.Private(); len(authed) > 0 {
		if err := e.signWsReq(ctx, baseReq); err != nil {
			errs = err
			for _, s := range authed {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", request.ErrAuthRequestFailed, s))
			}
			subs = subs.Public()
		}
	}

	for _, batch := range subs.GroupByPairs() {
		if baseReq.MessageType == subscribe && len(e.Websocket.GetSubscriptions()) != 0 {
			baseReq.MessageType = addSubscription // After first *successful* subscription API requires addSubscription
			baseReq.ClientType = clientType       // Note: Only addSubscription requires/accepts clientType
		}

		r := baseReq

		r.MarketIDs = batch[0].Pairs.Strings()
		r.Channels = make([]string, len(batch))
		for i, s := range batch {
			r.Channels[i] = s.QualifiedChannel
		}

		err := e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, r)
		if err == nil {
			err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, batch...)
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}

	return errs
}

func (e *Exchange) signWsReq(ctx context.Context, r *WsSubscribe) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	r.Timestamp = strconv.FormatInt(time.Now().UnixMilli(), 10)
	strToSign := "/users/self/subscribe" + "\n" + r.Timestamp
	tempSign, err := crypto.GetHMAC(crypto.HashSHA512, []byte(strToSign), []byte(creds.Secret))
	if err != nil {
		return err
	}
	r.Key = creds.Key
	r.Signature = base64.StdEncoding.EncodeToString(tempSign)
	return nil
}

// Unsubscribe sends a websocket message to manage and remove a subscription.
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	var errs error
	for _, s := range subs {
		req := WsSubscribe{
			MessageType: removeSubscription,
			ClientType:  clientType,
			Channels:    []string{s.Channel},
			MarketIDs:   s.Pairs.Strings(),
		}

		err := e.Websocket.Conn.SendJSONMessage(ctx, request.Unset, req)
		if err == nil {
			err = e.Websocket.RemoveSubscriptions(e.Websocket.Conn, s)
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// ReSubscribeSpecificOrderbook removes the subscription and the subscribes
// again to fetch a new snapshot in the event of a de-sync event.
func (e *Exchange) ReSubscribeSpecificOrderbook(pair currency.Pair) error {
	sub := subscription.List{{
		Channel: wsOrderbookUpdate,
		Pairs:   currency.Pairs{pair},
		Asset:   asset.Spot,
	}}
	if err := e.Unsubscribe(sub); err != nil && !errors.Is(err, subscription.ErrNotFound) {
		// ErrNotFound is okay, because we might be re-subscribing a single pair from a larger list
		// BTC-Market handles unsub/sub of one pair gracefully and the other pairs are unaffected
		return err
	}
	return e.Subscribe(sub)
}

// orderbookChecksum calculates a checksum for the orderbook liquidity
func orderbookChecksum(ob *orderbook.Book) uint32 {
	return crc32.ChecksumIEEE([]byte(concatOrderbookLiquidity(ob.Bids) + concatOrderbookLiquidity(ob.Asks)))
}

// concatOrderbookLiquidity concatenates price and amounts together for checksum processing
func concatOrderbookLiquidity(liquidity orderbook.Levels) string {
	var c strings.Builder
	for x := range min(10, len(liquidity)) {
		c.WriteString(trim(liquidity[x].Price))
		c.WriteString(trim(liquidity[x].Amount))
	}
	return c.String()
}

// trim turns value into string, removes the decimal point and all the leading zeros
func trim(value float64) string {
	valstr := strconv.FormatFloat(value, 'f', -1, 64)
	valstr = strings.ReplaceAll(valstr, ".", "")
	valstr = strings.TrimLeft(valstr, "0")
	return valstr
}

func channelName(s *subscription.Subscription) string {
	if n, ok := subscriptionNames[s.Channel]; ok {
		return n
	}
	panic(fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel))
}

const subTplText = `
{{ range $asset, $pairs := $.AssetPairs }}
	{{- channelName $.S -}}
	{{ $.AssetSeparator }}
{{- end }}
`
