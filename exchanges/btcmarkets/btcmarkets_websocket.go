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
func (b *BTCMarkets) WsConnect() error {
	ctx := context.TODO()
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	err := b.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return err
	}
	if b.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", b.Name)
	}

	b.Websocket.Wg.Add(1)
	go b.wsReadData(ctx)
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (b *BTCMarkets) wsReadData(ctx context.Context) {
	defer b.Websocket.Wg.Done()

	for {
		resp := b.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := b.wsHandleData(ctx, resp.Raw)
		if err != nil {
			b.Websocket.DataHandler <- err
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

func (b *BTCMarkets) wsHandleData(ctx context.Context, respRaw []byte) error {
	var wsResponse WsMessageType
	err := json.Unmarshal(respRaw, &wsResponse)
	if err != nil {
		return err
	}
	switch wsResponse.MessageType {
	case heartbeat:
		if b.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket heartbeat received %s", b.Name, respRaw)
		}
	case wsOrderbookUpdate:
		var ob WsOrderbook
		err := json.Unmarshal(respRaw, &ob)
		if err != nil {
			return err
		}

		if ob.Snapshot {
			err = b.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
				Pair:              ob.Currency,
				Bids:              orderbook.Levels(ob.Bids),
				Asks:              orderbook.Levels(ob.Asks),
				LastUpdated:       ob.Timestamp,
				LastUpdateID:      ob.SnapshotID,
				Asset:             asset.Spot,
				Exchange:          b.Name,
				ValidateOrderbook: b.ValidateOrderbook,
			})
		} else {
			err = b.Websocket.Orderbook.Update(&orderbook.Update{
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
				err2 := b.ReSubscribeSpecificOrderbook(ob.Currency)
				if err2 != nil {
					return err2
				}
			}
			return err
		}
		return nil
	case tradeEndPoint:
		tradeFeed := b.IsTradeFeedEnabled()
		saveTradeData := b.IsSaveTradeDataEnabled()
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
			Exchange:     b.Name,
			Price:        t.Price,
			Amount:       t.Volume,
			Side:         side,
			TID:          strconv.FormatInt(t.TradeID, 10),
		}

		if tradeFeed {
			b.Websocket.DataHandler <- td
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

		b.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: b.Name,
			Volume:       tick.Volume,
			High:         tick.High24,
			Low:          tick.Low24h,
			Bid:          tick.Bid,
			Ask:          tick.Ask,
			Last:         tick.Last,
			LastUpdated:  tick.Timestamp,
			AssetType:    asset.Spot,
			Pair:         tick.MarketID,
		}
	case fundChange:
		var transferData WsFundTransfer
		err := json.Unmarshal(respRaw, &transferData)
		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- transferData
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
				Exchange: b.Name,
				TID:      strconv.FormatInt(orderData.Trades[x].TradeID, 10),
				IsMaker:  isMaker,
			})
			price = orderData.Trades[x].Price
			originalAmount += orderData.Trades[x].Volume
		}
		oType, err := order.StringToOrderType(orderData.OrderType)
		if err != nil {
			b.Websocket.DataHandler <- order.ClassificationError{
				Exchange: b.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		oSide, err := order.StringToOrderSide(orderData.Side)
		if err != nil {
			b.Websocket.DataHandler <- order.ClassificationError{
				Exchange: b.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}
		oStatus, err := order.StringToOrderStatus(orderData.Status)
		if err != nil {
			b.Websocket.DataHandler <- order.ClassificationError{
				Exchange: b.Name,
				OrderID:  orderID,
				Err:      err,
			}
		}

		clientID := ""
		if creds, err := b.GetCredentials(ctx); err != nil {
			b.Websocket.DataHandler <- order.ClassificationError{
				Exchange: b.Name,
				OrderID:  orderID,
				Err:      err,
			}
		} else if creds != nil {
			clientID = creds.ClientID
		}

		b.Websocket.DataHandler <- &order.Detail{
			Price:           price,
			Amount:          originalAmount,
			RemainingAmount: orderData.OpenVolume,
			Exchange:        b.Name,
			OrderID:         orderID,
			ClientID:        clientID,
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       asset.Spot,
			Date:            orderData.Timestamp,
			Trades:          trades,
			Pair:            orderData.MarketID,
		}
	case "error":
		var wsErr WsError
		err := json.Unmarshal(respRaw, &wsErr)
		if err != nil {
			return err
		}
		return fmt.Errorf("%v websocket error. Code: %v Message: %v", b.Name, wsErr.Code, wsErr.Message)
	default:
		b.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: b.Name + websocket.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
}

func (b *BTCMarkets) generateSubscriptions() (subscription.List, error) {
	return b.Features.Subscriptions.ExpandTemplates(b)
}

// GetSubscriptionTemplate returns a subscription channel template
func (b *BTCMarkets) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{"channelName": channelName}).Parse(subTplText)
}

// Subscribe sends a websocket message to receive data from the channel
func (b *BTCMarkets) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	baseReq := &WsSubscribe{
		MessageType: subscribe,
	}

	var errs error
	if authed := subs.Private(); len(authed) > 0 {
		if err := b.signWsReq(ctx, baseReq); err != nil {
			errs = err
			for _, s := range authed {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", request.ErrAuthRequestFailed, s))
			}
			subs = subs.Public()
		}
	}

	for _, batch := range subs.GroupByPairs() {
		if baseReq.MessageType == subscribe && len(b.Websocket.GetSubscriptions()) != 0 {
			baseReq.MessageType = addSubscription // After first *successful* subscription API requires addSubscription
			baseReq.ClientType = clientType       // Note: Only addSubscription requires/accepts clientType
		}

		r := baseReq

		r.MarketIDs = batch[0].Pairs.Strings()
		r.Channels = make([]string, len(batch))
		for i, s := range batch {
			r.Channels[i] = s.QualifiedChannel
		}

		err := b.Websocket.Conn.SendJSONMessage(ctx, request.Unset, r)
		if err == nil {
			err = b.Websocket.AddSuccessfulSubscriptions(b.Websocket.Conn, batch...)
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}

	return errs
}

func (b *BTCMarkets) signWsReq(ctx context.Context, r *WsSubscribe) error {
	creds, err := b.GetCredentials(ctx)
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
func (b *BTCMarkets) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	var errs error
	for _, s := range subs {
		req := WsSubscribe{
			MessageType: removeSubscription,
			ClientType:  clientType,
			Channels:    []string{s.Channel},
			MarketIDs:   s.Pairs.Strings(),
		}

		err := b.Websocket.Conn.SendJSONMessage(ctx, request.Unset, req)
		if err == nil {
			err = b.Websocket.RemoveSubscriptions(b.Websocket.Conn, s)
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// ReSubscribeSpecificOrderbook removes the subscription and the subscribes
// again to fetch a new snapshot in the event of a de-sync event.
func (b *BTCMarkets) ReSubscribeSpecificOrderbook(pair currency.Pair) error {
	sub := subscription.List{{
		Channel: wsOrderbookUpdate,
		Pairs:   currency.Pairs{pair},
		Asset:   asset.Spot,
	}}
	if err := b.Unsubscribe(sub); err != nil && !errors.Is(err, subscription.ErrNotFound) {
		// ErrNotFound is okay, because we might be re-subscribing a single pair from a larger list
		// BTC-Market handles unsub/sub of one pair gracefully and the other pairs are unaffected
		return err
	}
	return b.Subscribe(sub)
}

// orderbookChecksum calculates a checksum for the orderbook liquidity
func orderbookChecksum(ob *orderbook.Book) uint32 {
	return crc32.ChecksumIEEE([]byte(concatOrderbookLiquidity(ob.Bids) + concatOrderbookLiquidity(ob.Asks)))
}

// concatOrderbookLiquidity concatenates price and amounts together for checksum processing
func concatOrderbookLiquidity(liquidity orderbook.Levels) string {
	var c string
	for x := range min(10, len(liquidity)) {
		c += trim(liquidity[x].Price) + trim(liquidity[x].Amount)
	}
	return c
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
